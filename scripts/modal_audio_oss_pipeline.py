#!/usr/bin/env python3
"""Modal-native OSS audio anti-spoofing data pipeline.

Runs dataset staging, manifest generation, local acoustic feature extraction, and
holdout reporting inside a Modal volume so large public corpora never need to be
stored in this workspace.
"""

from __future__ import annotations

import modal


APP_NAME = "mighty-morphing-audio-oss"
VOLUME_NAME = "citadel-audio-oss-data"
PREFIX = "/data/mighty-morphing/audio"

app = modal.App(APP_NAME)
volume = modal.Volume.from_name(VOLUME_NAME, create_if_missing=True)

image = (
    modal.Image.debian_slim(python_version="3.11")
    .apt_install("ca-certificates", "curl", "ffmpeg", "libsndfile1", "unzip")
    .pip_install("numpy>=1.26", "soundfile>=0.12", "huggingface_hub>=0.24", "requests>=2.31", "tqdm>=4.66")
    .add_local_dir("scripts", remote_path="/root/natsec/scripts")
)


DATASETS = {
    "in-the-wild": {
        "adapter": "in-the-wild",
        "dataset_name": "in_the_wild",
        "root": f"{PREFIX}/public/in_the_wild/release_in_the_wild",
        "zip": f"{PREFIX}/public/in_the_wild/release_in_the_wild.zip",
        "url": "https://huggingface.co/datasets/mueller91/In-The-Wild/resolve/main/release_in_the_wild.zip",
        "license": "cc-by-sa-4.0",
    },
    "fixtures-user": {
        "adapter": "fixtures-user",
        "dataset_name": "hampshire_user",
        "root": f"{PREFIX}/fixtures/user",
        "license": "private-local",
    },
}


@app.function(image=image, timeout=24 * 60 * 60, volumes={"/data": volume}, memory=32768)
def run_pipeline(dataset: str, max_per_label: int = 0, extract_features: bool = True, train_report: bool = True) -> dict[str, str]:
    import json
    import os
    import shlex
    import subprocess
    from pathlib import Path

    volume.reload()
    if dataset not in DATASETS:
        raise ValueError(f"unknown dataset {dataset!r}; choose one of {sorted(DATASETS)}")
    cfg = DATASETS[dataset]
    root = Path(cfg["root"])
    manifest = Path(f"{PREFIX}/manifests/{cfg['dataset_name']}.csv")
    features = Path(f"{PREFIX}/features/{cfg['dataset_name']}_local_features.csv")
    report = Path(f"{PREFIX}/reports/{cfg['dataset_name']}_local_holdouts.md")
    manifest.parent.mkdir(parents=True, exist_ok=True)
    features.parent.mkdir(parents=True, exist_ok=True)
    report.parent.mkdir(parents=True, exist_ok=True)

    if cfg.get("url"):
        stage_zip_dataset(cfg)

    build_cmd = [
        "python",
        "/root/natsec/scripts/build_audio_manifest.py",
        "--adapter",
        cfg["adapter"],
        "--root",
        str(root),
        "--out",
        str(manifest),
        "--dataset-name",
        cfg["dataset_name"],
        "--license",
        cfg["license"],
    ]
    if max_per_label > 0:
        build_cmd += ["--max-per-label", str(max_per_label)]
    run(build_cmd)

    result = {"dataset": dataset, "manifest": str(manifest)}
    if extract_features:
        run(
            [
                "python",
                "/root/natsec/scripts/extract_spoof_features.py",
                "--manifest",
                str(manifest),
                "--skip-voicebio",
                "--skip-citadel",
                "--out",
                str(features),
            ]
        )
        result["features"] = str(features)
    elif features.exists():
        result["features"] = str(features)

    if train_report:
        if not features.exists():
            raise ValueError(f"train_report requires feature extraction or an existing feature file: {features}")
        run(
            [
                "python",
                "/root/natsec/scripts/train_spoof_meta.py",
                "--features",
                str(features),
                "--out",
                str(report),
                "--eval-mode",
                "split",
                "--test-size",
                "0.25",
                "--seed",
                "7",
                "--group-holdout-mode",
                "split",
                "--group-by",
                "dataset",
                "--group-by",
                "generator_id",
                "--group-by",
                "speaker_id",
            ]
        )
        result["report"] = str(report)

    result["summary"] = summarize_outputs(manifest, features if extract_features else None, report if train_report else None)
    print(json.dumps(result, indent=2))
    volume.commit()
    return result


@app.function(image=image, timeout=60 * 60, volumes={"/data": volume}, memory=8192)
def list_audio_tree(path: str = PREFIX) -> list[str]:
    import os

    volume.reload()
    rows = []
    for root, dirs, files in os.walk(path):
        dirs[:] = sorted(dirs)
        files = sorted(files)
        rows.append(root)
        for name in files[:20]:
            rows.append(os.path.join(root, name))
        if len(files) > 20:
            rows.append(os.path.join(root, f"... {len(files) - 20} more files"))
        if len(rows) > 300:
            rows.append("... truncated")
            break
    for row in rows:
        print(row)
    return rows


def stage_zip_dataset(cfg: dict[str, str]) -> None:
    from pathlib import Path

    zip_path = Path(cfg["zip"])
    root = Path(cfg["root"])
    root.parent.mkdir(parents=True, exist_ok=True)
    marker = root / ".staged"
    if marker.exists() and any(root.rglob("*")):
        print(f"Using existing staged dataset: {root}")
        return
    if not zip_path.exists():
        run(["curl", "-L", "--fail", "--retry", "5", "--silent", "--show-error", "-o", str(zip_path), cfg["url"]])
    root.mkdir(parents=True, exist_ok=True)
    run(["unzip", "-q", "-o", str(zip_path), "-d", str(root)])
    marker.write_text("ok\n")


def summarize_outputs(manifest, features, report) -> str:
    from pathlib import Path

    chunks = []
    for path in [manifest, features, report]:
        if path is None:
            continue
        p = Path(path)
        if p.exists():
            chunks.append(f"{p}: {p.stat().st_size} bytes")
    return "; ".join(chunks)


def run(cmd: list[str]) -> None:
    import shlex
    import subprocess

    print("+ " + " ".join(shlex.quote(part) for part in cmd), flush=True)
    subprocess.run(cmd, check=True)


@app.local_entrypoint()
def main(dataset: str = "fixtures-user", max_per_label: int = 0, extract_features: bool = True, train_report: bool = True, list_only: bool = False):
    if list_only:
        list_audio_tree.remote()  # pyright: ignore[reportFunctionMemberAccess]
        return
    result = run_pipeline.remote(dataset, max_per_label, extract_features, train_report)  # pyright: ignore[reportFunctionMemberAccess]
    print(result)
