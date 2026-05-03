#!/usr/bin/env python3
"""Run Phase 0 detector validation over fixtures/user audio.

The harness writes:
  docs/detector_validation_results.csv
  docs/DETECTOR_VALIDATION.md
"""

from __future__ import annotations

import argparse
import csv
import json
import math
import mimetypes
import os
import statistics
import sys
import time
import urllib.error
import urllib.request
import uuid
from pathlib import Path
from typing import Any


AUDIO_SUFFIXES = {".wav", ".mp3", ".m4a", ".flac", ".ogg", ".aiff", ".aif"}
CSV_FIELDS = [
    "file",
    "source",
    "label",
    "duration_s",
    "embed_cosine_vs_real_01",
    "citadel_action",
    "citadel_risk",
    "citadel_deepfake_risk",
    "citadel_injection_risk",
    "citadel_processing_ms",
    "elevenlabs_classifier_ai_score",
    "notes",
]


def main() -> int:
    load_dotenv()
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--fixtures-dir", default="fixtures/user")
    parser.add_argument("--docs-dir", default="docs")
    parser.add_argument("--voicebio-url", default=os.getenv("VOICEBIO_URL", "http://localhost:7100"))
    parser.add_argument("--citadel-audio-url", default=os.getenv("CITADEL_AUDIO_URL", ""))
    parser.add_argument("--citadel-api-key", default=os.getenv("CITADEL_API_KEY", ""))
    parser.add_argument("--mode", default="secure")
    args = parser.parse_args()

    fixtures_dir = Path(args.fixtures_dir)
    docs_dir = Path(args.docs_dir)
    docs_dir.mkdir(parents=True, exist_ok=True)
    csv_path = docs_dir / "detector_validation_results.csv"
    report_path = docs_dir / "DETECTOR_VALIDATION.md"

    files = inventory(fixtures_dir)
    if not files:
        write_empty_report(report_path, fixtures_dir, "No Phase 0 audio fixtures found.")
        return 2
    baseline = next((p for p in files if p.stem == "real_01"), None)
    if baseline is None:
        write_empty_report(report_path, fixtures_dir, "Missing real_01.* baseline recording.")
        return 2

    rows: list[dict[str, Any]] = []
    embeddings: dict[Path, list[float]] = {}
    embed_meta: dict[Path, dict[str, Any]] = {}
    notes: dict[Path, list[str]] = {p: [] for p in files}

    for path in files:
        try:
            embed = post_audio_json(args.voicebio_url.rstrip("/") + "/embed", path, field="audio")
            embeddings[path] = [float(x) for x in embed["embedding"]]
            embed_meta[path] = embed
        except Exception as e:  # noqa: BLE001
            notes[path].append(f"voicebio error: {e}")

    baseline_emb = embeddings.get(baseline)
    for path in files:
        row: dict[str, Any] = {
            "file": str(path),
            "source": source_for(path),
            "label": label_for(path),
            "duration_s": "",
            "embed_cosine_vs_real_01": "",
            "citadel_action": "",
            "citadel_risk": "",
            "citadel_deepfake_risk": "",
            "citadel_injection_risk": "",
            "citadel_processing_ms": "",
            "elevenlabs_classifier_ai_score": "",
            "notes": "",
        }
        if path in embeddings and baseline_emb:
            row["embed_cosine_vs_real_01"] = f"{cosine(baseline_emb, embeddings[path]):.6f}"
        if path in embed_meta:
            row["duration_s"] = f"{float(embed_meta[path].get('duration_s', 0)):.3f}"

        if args.citadel_audio_url:
            try:
                scan = post_audio_json(
                    args.citadel_audio_url.rstrip() + "/v1/audio/scan/file",
                    path,
                    field="file",
                    fields={"mode": args.mode},
                    headers=citadel_headers(args.citadel_api_key),
                )
                row["citadel_action"] = scan.get("action", "")
                row["citadel_risk"] = risk_value(scan, "risk", "risk_score")
                row["citadel_deepfake_risk"] = risk_value(scan, "deepfake_risk", signal="deepfake_detection", score="deepfake_score")
                row["citadel_injection_risk"] = risk_value(scan, "injection_risk", signal="asr_prefix_attack")
                row["citadel_processing_ms"] = risk_value(scan, "processing_ms", "processing_time_ms")
            except Exception as e:  # noqa: BLE001
                notes[path].append(f"citadel error: {e}")
        else:
            notes[path].append("CITADEL_AUDIO_URL not set")

        row["notes"] = "; ".join(notes[path])
        rows.append(row)

    with csv_path.open("w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=CSV_FIELDS)
        writer.writeheader()
        writer.writerows(rows)
    write_report(report_path, rows, args, embeddings)
    print(f"wrote {csv_path}")
    print(f"wrote {report_path}")
    return 0


def inventory(fixtures_dir: Path) -> list[Path]:
    if not fixtures_dir.exists():
        return []
    return sorted(p for p in fixtures_dir.iterdir() if p.is_file() and p.suffix.lower() in AUDIO_SUFFIXES)


def source_for(path: Path) -> str:
    stem = path.stem
    if stem.startswith("real_"):
        return "real"
    if stem.startswith("clone_cartesia_"):
        return "cartesia"
    if stem.startswith("clone_elevenlabs_"):
        return "elevenlabs"
    if stem.startswith("clone_"):
        return "clone"
    return "control"


def label_for(path: Path) -> str:
    return "real" if source_for(path) == "real" else "synthetic"


def write_report(path: Path, rows: list[dict[str, Any]], args: argparse.Namespace, embeddings: dict[Path, list[float]]) -> None:
    real = [r for r in rows if r["label"] == "real"]
    synth = [r for r in rows if r["label"] == "synthetic"]
    real_df = numeric(real, "citadel_deepfake_risk")
    synth_df = numeric(synth, "citadel_deepfake_risk")
    real_cos = numeric(real, "embed_cosine_vs_real_01")
    synth_cos = numeric(synth, "embed_cosine_vs_real_01")
    verdict = verdict_for(real_df, synth_df, real_cos, rows)
    lines = [
        "# Detector Validation",
        "",
        f"Generated: {utc_now()}",
        "",
        "## Sample Inventory",
        "",
        "| file | source | label | cosine vs real_01 | Citadel deepfake | notes |",
        "|---|---:|---:|---:|---:|---|",
    ]
    for r in rows:
        lines.append(
            f"| `{Path(str(r['file'])).name}` | {r['source']} | {r['label']} | {display_value(r['embed_cosine_vs_real_01'])} | {display_value(r['citadel_deepfake_risk'])} | {r['notes'] or ''} |"
        )
    lines.extend(
        [
            "",
            "## Distributions",
            "",
            f"- Real deepfake risk: {summary(real_df)}",
            f"- Clone deepfake risk: {summary(synth_df)}",
            f"- Real overall Citadel risk: {summary(numeric(real, 'citadel_risk'))}",
            f"- Clone overall Citadel risk: {summary(numeric(synth, 'citadel_risk'))}",
            f"- Real cosine vs real_01: {summary(real_cos)}",
            f"- Clone/control cosine vs real_01: {summary(synth_cos)}",
            "",
            "## Speaker Baseline Analysis",
            "",
            speaker_analysis(rows, embeddings),
            "",
            "## Verdict",
            "",
            verdict,
            "",
            "## Threshold Recommendation",
            "",
            threshold_recommendation(real_df, synth_df, real_cos),
            "",
            "## Reproduction",
            "",
            "```bash",
            "cd natsec",
            "python3 scripts/clone_voice.py",
            "python3 scripts/validate_detector.py",
            "```",
            "",
            "## Configuration",
            "",
            f"- voicebio: `{args.voicebio_url}`",
            f"- citadel_audio: `{args.citadel_audio_url or 'not configured'}`",
            f"- mode: `{args.mode}`",
        ]
    )
    path.write_text("\n".join(lines) + "\n")


def write_empty_report(path: Path, fixtures_dir: Path, reason: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        "\n".join(
            [
                "# Detector Validation",
                "",
                f"Generated: {utc_now()}",
                "",
                "Status: BLOCKED",
                "",
                reason,
                "",
                "Required files:",
                "",
                f"- `{fixtures_dir}/real_01.wav`",
                f"- `{fixtures_dir}/real_02.wav`",
                f"- `{fixtures_dir}/real_03.wav`",
                "",
            ]
        )
    )


def verdict_for(real_df: list[float], synth_df: list[float], real_cos: list[float], rows: list[dict[str, Any]]) -> str:
    has_500 = any("citadel error" in str(r.get("notes", "")) for r in rows)
    if real_df and synth_df and statistics.mean(real_df) < 30 and statistics.mean(synth_df) > 70 and real_cos and min(real_cos) > 0.85 and not has_500:
        return "PASS: real and clone distributions are cleanly separated under the plan rule."
    if real_df or synth_df:
        return "FAIL for passive deepfake separation: measured values exist, but clone scores are not higher than real scores under the plan rule. Reframe the demo around OOB as the decisive factor."
    return "FAIL/BLOCKED: detector results are missing. Do not claim passive separation until this harness has measured it."


def threshold_recommendation(real_df: list[float], synth_df: list[float], real_cos: list[float]) -> str:
    parts: list[str] = []
    if real_df and synth_df:
        real_p95 = percentile(real_df, 95)
        synth_p05 = percentile(synth_df, 5)
        if real_p95 < synth_p05:
            parts.append(f"- `MM_DEEPFAKE_BLOCK_THRESHOLD={int((real_p95 + synth_p05) / 2)}` from midpoint between real p95 and clone p05.")
        else:
            parts.append("- Deepfake threshold not separable at 95/5; keep conservative default and rely on OOB.")
    if real_cos:
        parts.append(f"- `MM_SPEAKER_MATCH_THRESHOLD={max(0.0, percentile(real_cos, 5) - 0.02):.2f}` from real-sample p05 minus margin.")
    return "\n".join(parts) if parts else "No threshold recommendation until real and clone measurements exist."


def risk_value(scan: dict[str, Any], *keys: str, signal: str | None = None, score: str | None = None) -> int | str:
    for key in keys:
        value = scan.get(key)
        if value not in ("", None):
            return normalize_risk(value)
    if signal:
        for item in scan.get("signals") or []:
            if isinstance(item, dict) and item.get("name") == signal:
                return normalize_risk(item.get("risk_score"))
    if score and scan.get(score) not in ("", None):
        return normalize_risk(scan.get(score))
    return ""


def normalize_risk(value: Any) -> int:
    try:
        f = float(value)
    except (TypeError, ValueError):
        return 0
    if 0 <= f <= 1:
        return int(round(f * 100))
    return int(round(f))


def display_value(value: Any) -> str:
    if value == "" or value is None:
        return "-"
    return str(value)


def speaker_analysis(rows: list[dict[str, Any]], embeddings: dict[Path, list[float]]) -> str:
    real_paths = [Path(str(row["file"])) for row in rows if row["label"] == "real" and Path(str(row["file"])) in embeddings]
    if len(real_paths) < 2:
        return "Need at least two real embeddings for speaker range analysis."
    centroid = vector_mean([embeddings[p] for p in real_paths])
    lines = ["| file | cosine vs real_01 | cosine vs real centroid |", "|---|---:|---:|"]
    for path in real_paths:
        real_01 = embeddings[real_paths[0]]
        lines.append(f"| `{path.name}` | {cosine(real_01, embeddings[path]):.3f} | {cosine(centroid, embeddings[path]):.3f} |")
    pairwise = []
    for i, a in enumerate(real_paths):
        for b in real_paths[i + 1 :]:
            pairwise.append(cosine(embeddings[a], embeddings[b]))
    lines.append("")
    lines.append(f"Pairwise real-vs-real cosine: {summary(pairwise)}")
    lines.append("")
    lines.append("Interpretation: single-anchor matching is sensitive to modulation and recording condition. A production enrollment should store multiple templates or a centroid plus outlier policy, not one embedding.")
    return "\n".join(lines)


def vector_mean(vectors: list[list[float]]) -> list[float]:
    if not vectors:
        return []
    dim = len(vectors[0])
    out = [0.0] * dim
    for vec in vectors:
        for i, value in enumerate(vec):
            out[i] += value
    return [value / len(vectors) for value in out]


def numeric(rows: list[dict[str, Any]], key: str) -> list[float]:
    vals: list[float] = []
    for row in rows:
        raw = row.get(key)
        if raw in ("", None):
            continue
        try:
            vals.append(float(raw))
        except (TypeError, ValueError):
            pass
    return vals


def summary(vals: list[float]) -> str:
    if not vals:
        return "n=0"
    return f"n={len(vals)}, mean={statistics.mean(vals):.2f}, p50={percentile(vals, 50):.2f}, p95={percentile(vals, 95):.2f}"


def percentile(vals: list[float], p: float) -> float:
    if not vals:
        return math.nan
    ordered = sorted(vals)
    if len(ordered) == 1:
        return ordered[0]
    rank = (len(ordered) - 1) * (p / 100)
    lo = math.floor(rank)
    hi = math.ceil(rank)
    if lo == hi:
        return ordered[int(rank)]
    return ordered[lo] + (ordered[hi] - ordered[lo]) * (rank - lo)


def cosine(a: list[float], b: list[float]) -> float:
    if len(a) != len(b) or not a:
        return 0.0
    dot = sum(x * y for x, y in zip(a, b, strict=True))
    na = math.sqrt(sum(x * x for x in a))
    nb = math.sqrt(sum(y * y for y in b))
    if na <= 1e-9 or nb <= 1e-9:
        return 0.0
    return dot / (na * nb)


def post_audio_json(url: str, path: Path, field: str, fields: dict[str, str] | None = None, headers: dict[str, str] | None = None) -> dict[str, Any]:
    boundary = "----mm-" + uuid.uuid4().hex
    chunks: list[bytes] = []
    for name, value in (fields or {}).items():
        chunks.append(f"--{boundary}\r\n".encode())
        chunks.append(f'Content-Disposition: form-data; name="{name}"\r\n\r\n'.encode())
        chunks.append(str(value).encode())
        chunks.append(b"\r\n")
    filename = path.name
    content_type = mimetypes.guess_type(filename)[0] or "application/octet-stream"
    chunks.append(f"--{boundary}\r\n".encode())
    chunks.append(f'Content-Disposition: form-data; name="{field}"; filename="{filename}"\r\n'.encode())
    chunks.append(f"Content-Type: {content_type}\r\n\r\n".encode())
    chunks.append(path.read_bytes())
    chunks.append(b"\r\n")
    chunks.append(f"--{boundary}--\r\n".encode())

    req = urllib.request.Request(url, data=b"".join(chunks), method="POST")
    req.add_header("Content-Type", f"multipart/form-data; boundary={boundary}")
    for key, value in (headers or {}).items():
        req.add_header(key, value)
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:  # noqa: S310 - local/configured service
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8", "replace")
        raise RuntimeError(f"HTTP {e.code}: {body[:400]}") from e


def citadel_headers(api_key: str) -> dict[str, str]:
    if not api_key:
        return {}
    return {"X-API-Key": api_key, "Authorization": f"Bearer {api_key}", "X-Internal-Token": api_key}


def utc_now() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def load_dotenv(path: str = ".env") -> None:
    env_path = Path(path)
    if not env_path.exists():
        return
    for raw in env_path.read_text().splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        key = key.strip()
        value = value.split("#", 1)[0].strip().strip('"').strip("'")
        if key and key not in os.environ:
            os.environ[key] = value


if __name__ == "__main__":
    raise SystemExit(main())
