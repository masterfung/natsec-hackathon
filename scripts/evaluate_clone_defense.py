#!/usr/bin/env python3
"""Evaluate controlled real/clone fixtures against the deployed audio detector.

This is a defensive harness. It scans locally owned/consented real voice
fixtures, controlled Cartesia clone fixtures, and existing ElevenLabs benchmark
fixtures when present. It does not create clones; generation lives in
clone_voice.py and requires provider keys plus explicit consent for the source
voice.
"""

from __future__ import annotations

import argparse
import json
import os
import time
from pathlib import Path
from typing import Any

import requests


DEFAULT_ENDPOINT = "http://localhost:8001/v1/audio/scan/file"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--endpoint", default=os.getenv("CITADEL_AUDIO_SCAN_URL", DEFAULT_ENDPOINT))
    parser.add_argument("--out", default="docs/clone_defense_eval_live.json")
    parser.add_argument("--real-limit", type=int, default=3)
    parser.add_argument("--cartesia-limit", type=int, default=6)
    parser.add_argument("--elevenlabs-limit", type=int, default=6)
    args = parser.parse_args()

    load_dotenv()
    api_key = os.getenv("CITADEL_API_KEY", "").strip()
    if not api_key:
        raise SystemExit("CITADEL_API_KEY missing")

    root = Path(__file__).resolve().parents[1]
    repo = root.parent
    items: list[tuple[str, Path]] = []
    user_fixtures = root / "fixtures/user"
    items.extend(("real_holdout", p) for p in sorted(user_fixtures.glob("real_*.wav"))[: args.real_limit])
    items.extend(("cartesia_clone", p) for p in sorted(user_fixtures.glob("clone_cartesia_*.wav"))[: args.cartesia_limit])

    eleven_dir = repo / "tests/artifacts/audio/deepfake_elevenlabs"
    if eleven_dir.exists():
        items.extend(("elevenlabs_deepfake", p) for p in sorted(eleven_dir.glob("*.*"))[: args.elevenlabs_limit])
    fresh_dir = repo / ".context/fresh-clone-eval"
    if fresh_dir.exists():
        items.extend(("fresh_cartesia_redteam", p) for p in sorted(fresh_dir.glob("clone_cartesia_*.wav"))[:1])

    rows = [scan_file(args.endpoint, api_key, label, path, root, repo) for label, path in items]
    result = {
        "created_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "endpoint": args.endpoint,
        "policy": "Controlled defensive evaluation only. Use consented source voices for provider-generated clone fixtures.",
        "summary": summarize(rows),
        "rows": rows,
    }
    out = Path(args.out)
    if not out.is_absolute():
        out = root / out
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps(result["summary"], indent=2))
    print(f"wrote {out}")
    return 0


def scan_file(endpoint: str, api_key: str, label: str, path: Path, root: Path, repo: Path) -> dict[str, Any]:
    with path.open("rb") as fh:
        res = requests.post(
            endpoint,
            headers={"X-API-Key": api_key},
            files={"file": (path.name, fh, content_type(path))},
            data={"mode": "secure", "direction": "input"},
            timeout=90,
        )
    row: dict[str, Any] = {"label": label, "file": display_path(path, root, repo), "status_code": res.status_code}
    if res.status_code != 200:
        row["error"] = res.text[:600]
        return row

    data = res.json()
    signals = {s.get("name"): s for s in data.get("signals", []) if isinstance(s, dict)}
    row.update(
        {
            "action": data.get("action"),
            "risk_score": data.get("risk_score"),
            "risk_level": data.get("risk_level"),
            "processing_time_ms": data.get("processing_time_ms"),
            "deepfake_score": data.get("deepfake_score"),
            "deepfake_signal": signal_score(signals, "deepfake_detection"),
            "wavlm_signal": signal_score(signals, "wavlm_paralinguistic"),
            "rmt_signal": signal_score(signals, "rmt_divergence"),
            "text_signal": signal_score(signals, "text_pipeline"),
        }
    )
    print(
        f"{label:20s} {path.name:32s} {row.get('action')} risk={row.get('risk_score')} "
        f"df={row.get('deepfake_signal')} wl={row.get('wavlm_signal')} ms={row.get('processing_time_ms')}"
    )
    return row


def display_path(path: Path, root: Path, repo: Path) -> str:
    """Store portable paths in reports instead of workstation absolute paths."""
    resolved = path.resolve()
    for base, prefix in ((root.resolve(), ""), (repo.resolve(), "../")):
        try:
            rel = resolved.relative_to(base)
        except ValueError:
            continue
        return f"{prefix}{rel.as_posix()}"
    return path.name


def signal_score(signals: dict[str, Any], name: str) -> Any:
    sig = signals.get(name) or {}
    return sig.get("risk_score") or sig.get("score")


def summarize(rows: list[dict[str, Any]]) -> dict[str, dict[str, Any]]:
    out: dict[str, dict[str, Any]] = {}
    for label in sorted({str(r["label"]) for r in rows}):
        group = [r for r in rows if r.get("label") == label and r.get("status_code") == 200]
        positives = [] if label == "real_holdout" else group
        out[label] = {
            "n": len(group),
            "allow": sum(1 for r in group if r.get("action") == "ALLOW"),
            "warn": sum(1 for r in group if r.get("action") == "WARN"),
            "block": sum(1 for r in group if r.get("action") == "BLOCK"),
            "detected_warn_or_block": (
                sum(1 for r in positives if r.get("action") in {"WARN", "BLOCK"}) if positives else None
            ),
        }
    return out


def content_type(path: Path) -> str:
    if path.suffix.lower() == ".mp3":
        return "audio/mpeg"
    return "audio/wav"


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
