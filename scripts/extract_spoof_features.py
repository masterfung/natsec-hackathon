#!/usr/bin/env python3
"""Extract passive spoof-detection features for real/clone audio fixtures."""

from __future__ import annotations

import argparse
import csv
import json
import math
import mimetypes
import os
import time
import urllib.error
import urllib.request
import uuid
from pathlib import Path
from typing import Any

import numpy as np
import soundfile as sf


AUDIO_SUFFIXES = {".wav", ".mp3", ".m4a", ".flac", ".ogg", ".aiff", ".aif"}
METADATA_FIELDS = [
    "dataset",
    "speaker_id",
    "language",
    "generator_id",
    "attack_type",
    "codec",
    "sample_rate",
    "license",
    "split",
    "source_url",
]
FEATURE_FIELDS = [
    "file",
    "source",
    "label",
    *METADATA_FIELDS,
    "duration_s",
    "speaker_cosine_vs_real_01",
    "citadel_action",
    "citadel_risk",
    "citadel_latency_ms",
    "sig_spectral_anomaly",
    "sig_steganalysis",
    "sig_ultrasonic_injection",
    "sig_wavlm_paralinguistic",
    "sig_deepfake_detection",
    "sig_waveform_integrity",
    "sig_rmt_divergence",
    "sig_asr_prefix_attack",
    "rms_mean",
    "rms_std",
    "silence_ratio",
    "centroid_mean",
    "centroid_std",
    "flatness_mean",
    "flatness_std",
    "hf_ratio_mean",
    "zcr_mean",
    "pitch_mean",
    "pitch_std",
    "pitch_cv",
]


def main() -> int:
    load_dotenv()
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--fixtures-dir", default="fixtures/user")
    parser.add_argument("--manifest", default="", help="Optional normalized manifest CSV from build_audio_manifest.py.")
    parser.add_argument("--baseline-audio", default="", help="Speaker baseline audio. Defaults to real_01 in fixtures/manifest.")
    parser.add_argument("--out", default="docs/spoof_features.csv")
    parser.add_argument("--voicebio-url", default=os.getenv("VOICEBIO_URL", "http://localhost:7100"))
    parser.add_argument("--citadel-audio-url", default=os.getenv("CITADEL_AUDIO_URL", ""))
    parser.add_argument("--citadel-api-key", default=os.getenv("CITADEL_API_KEY", ""))
    parser.add_argument("--mode", default="secure")
    parser.add_argument("--skip-voicebio", action="store_true")
    parser.add_argument("--skip-citadel", action="store_true")
    args = parser.parse_args()

    items = load_audio_items(args)
    if not items:
        raise SystemExit("no audio files to process")
    baseline = baseline_audio(items, args)
    if not args.skip_citadel and not args.citadel_audio_url:
        raise SystemExit("missing --citadel-audio-url; use --skip-citadel for local-only extraction")

    embeddings: dict[Path, list[float]] = {}
    baseline_emb: list[float] | None = None
    if not args.skip_voicebio:
        if baseline is None:
            raise SystemExit("missing speaker baseline; pass --baseline-audio or use --skip-voicebio")
        for item in items:
            path = item["path"]
            embed = post_audio_json(args.voicebio_url.rstrip("/") + "/embed", path, "audio")
            embeddings[path] = [float(x) for x in embed["embedding"]]
        baseline_emb = embeddings.get(baseline)
        if baseline_emb is None:
            embed = post_audio_json(args.voicebio_url.rstrip("/") + "/embed", baseline, "audio")
            baseline_emb = [float(x) for x in embed["embedding"]]

    rows: list[dict[str, Any]] = []
    for item in items:
        path = item["path"]
        row = {
            "file": str(path),
            "source": item.get("source") or item.get("generator_id") or item.get("dataset") or source_for(path),
            "label": item.get("label") or label_for(path),
            "duration_s": round(audio_duration(path), 3),
            "speaker_cosine_vs_real_01": round(cosine(baseline_emb, embeddings[path]), 6) if baseline_emb is not None else 0,
        }
        for field in METADATA_FIELDS:
            row[field] = item.get(field, "")
        row.update(local_audio_features(path))
        if not args.skip_citadel:
            scan = post_audio_json(
                args.citadel_audio_url.rstrip("/") + "/v1/audio/scan/file",
                path,
                "file",
                fields={"mode": args.mode},
                headers=citadel_headers(args.citadel_api_key),
                timeout=90,
            )
            row.update(citadel_features(scan))
        rows.append(row)

    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    with out_path.open("w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=FEATURE_FIELDS)
        writer.writeheader()
        writer.writerows(rows)
    print(f"wrote {out_path}")
    return 0


def load_audio_items(args: argparse.Namespace) -> list[dict[str, Any]]:
    if args.manifest:
        return manifest_items(Path(args.manifest))
    return [
        {
            "path": path,
            "source": source_for(path),
            "label": label_for(path),
            "dataset": "fixtures_user",
            "split": "eval",
            "codec": path.suffix.lower().lstrip("."),
        }
        for path in inventory(Path(args.fixtures_dir))
    ]


def manifest_items(manifest_path: Path) -> list[dict[str, Any]]:
    items: list[dict[str, Any]] = []
    with manifest_path.open(newline="") as f:
        for row in csv.DictReader(f):
            path = resolve_manifest_path(row.get("path", ""), manifest_path)
            if not path.is_file():
                continue
            item: dict[str, Any] = {field: row.get(field, "") for field in METADATA_FIELDS}
            item.update(
                {
                    "path": path,
                    "source": row.get("source", "") or row.get("generator_id", "") or row.get("dataset", ""),
                    "label": row.get("label", ""),
                    "dataset": row.get("dataset", ""),
                }
            )
            items.append(item)
    return items


def resolve_manifest_path(value: str, manifest_path: Path) -> Path:
    path = Path(value).expanduser()
    if path.is_absolute() and path.exists():
        return path
    cwd_path = Path.cwd() / path
    if cwd_path.exists():
        return cwd_path
    return manifest_path.parent / path


def baseline_audio(items: list[dict[str, Any]], args: argparse.Namespace) -> Path | None:
    if args.baseline_audio:
        return Path(args.baseline_audio).expanduser()
    return next((item["path"] for item in items if item["path"].name == "real_01.wav" or item["path"].stem == "real_01"), None)


def inventory(fixtures_dir: Path) -> list[Path]:
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


def audio_duration(path: Path) -> float:
    audio, sr = sf.read(path, dtype="float32")
    return len(audio) / sr


def local_audio_features(path: Path) -> dict[str, float]:
    audio, sr = sf.read(path, dtype="float32")
    if audio.ndim > 1:
        audio = audio.mean(axis=1)
    audio = audio / (float(np.max(np.abs(audio))) + 1e-9)
    frames = frame_audio(audio, sr)
    if len(frames) == 0:
        frames = audio.reshape(1, -1)
    rms = np.sqrt(np.mean(frames**2, axis=1) + 1e-12)
    voiced = rms > np.percentile(rms, 35) if len(rms) > 1 else np.ones_like(rms, dtype=bool)
    voiced_frames = frames[voiced]
    if len(voiced_frames) == 0:
        voiced_frames = frames
    win = np.hanning(voiced_frames.shape[1])
    spec = np.abs(np.fft.rfft(voiced_frames * win, axis=1)) + 1e-12
    freqs = np.fft.rfftfreq(voiced_frames.shape[1], 1 / sr)
    power = spec**2
    centroid = (power * freqs).sum(axis=1) / (power.sum(axis=1) + 1e-12)
    flatness = np.exp(np.mean(np.log(spec), axis=1)) / (np.mean(spec, axis=1) + 1e-12)
    hf = power[:, freqs > 4000].sum(axis=1) / (power.sum(axis=1) + 1e-12)
    zcr = np.mean(np.abs(np.diff(np.signbit(voiced_frames), axis=1)), axis=1)
    pitches = pitch_track(voiced_frames, sr)
    return {
        "rms_mean": round(float(np.mean(rms)), 6),
        "rms_std": round(float(np.std(rms)), 6),
        "silence_ratio": round(float(np.mean(rms < 0.01)), 6),
        "centroid_mean": round(float(np.mean(centroid)), 3),
        "centroid_std": round(float(np.std(centroid)), 3),
        "flatness_mean": round(float(np.mean(flatness)), 6),
        "flatness_std": round(float(np.std(flatness)), 6),
        "hf_ratio_mean": round(float(np.mean(hf)), 6),
        "zcr_mean": round(float(np.mean(zcr)), 6),
        "pitch_mean": round(float(np.mean(pitches)), 3),
        "pitch_std": round(float(np.std(pitches)), 3),
        "pitch_cv": round(float(np.std(pitches) / (np.mean(pitches) + 1e-9)), 6),
    }


def frame_audio(audio: np.ndarray, sr: int, ms: int = 25, hop_ms: int = 10) -> np.ndarray:
    n = int(sr * ms / 1000)
    hop = int(sr * hop_ms / 1000)
    if len(audio) < n:
        return np.empty((0, n))
    return np.stack([audio[i : i + n] for i in range(0, len(audio) - n + 1, hop)])


def pitch_track(frames: np.ndarray, sr: int) -> np.ndarray:
    pitches: list[float] = []
    min_lag = int(sr / 450)
    max_lag = int(sr / 60)
    step = max(1, len(frames) // 100)
    for frame in frames[::step]:
        frame = frame - frame.mean()
        ac = np.correlate(frame, frame, mode="full")[len(frame) - 1 :]
        if ac[0] <= 1e-9 or len(ac) <= max_lag:
            continue
        seg = ac[min_lag:max_lag]
        lag = int(np.argmax(seg) + min_lag)
        conf = float(seg[lag - min_lag] / ac[0])
        if conf > 0.25:
            pitches.append(sr / lag)
    return np.asarray(pitches or [0.0], dtype=np.float32)


def citadel_features(scan: dict[str, Any]) -> dict[str, Any]:
    signals = {item.get("name"): item.get("risk_score", 0) for item in scan.get("signals") or [] if isinstance(item, dict)}
    return {
        "citadel_action": scan.get("action", ""),
        "citadel_risk": normalize_risk(scan.get("risk_score", scan.get("risk", 0))),
        "citadel_latency_ms": normalize_risk(scan.get("processing_time_ms", scan.get("processing_ms", 0))),
        "sig_spectral_anomaly": normalize_risk(signals.get("spectral_anomaly", 0)),
        "sig_steganalysis": normalize_risk(signals.get("steganalysis", 0)),
        "sig_ultrasonic_injection": normalize_risk(signals.get("ultrasonic_injection", 0)),
        "sig_wavlm_paralinguistic": normalize_risk(signals.get("wavlm_paralinguistic", 0)),
        "sig_deepfake_detection": normalize_risk(signals.get("deepfake_detection", 0)),
        "sig_waveform_integrity": normalize_risk(signals.get("waveform_integrity", 0)),
        "sig_rmt_divergence": normalize_risk(signals.get("rmt_divergence", 0)),
        "sig_asr_prefix_attack": normalize_risk(signals.get("asr_prefix_attack", 0)),
    }


def normalize_risk(value: Any) -> int:
    try:
        f = float(value)
    except (TypeError, ValueError):
        return 0
    if 0 <= f <= 1:
        return int(round(f * 100))
    return int(round(f))


def cosine(a: list[float], b: list[float]) -> float:
    if len(a) != len(b) or not a:
        return 0.0
    dot = sum(x * y for x, y in zip(a, b, strict=True))
    na = math.sqrt(sum(x * x for x in a))
    nb = math.sqrt(sum(y * y for y in b))
    if na <= 1e-9 or nb <= 1e-9:
        return 0.0
    return dot / (na * nb)


def post_audio_json(url: str, path: Path, field: str, fields: dict[str, str] | None = None, headers: dict[str, str] | None = None, timeout: int = 60) -> dict[str, Any]:
    boundary = "----mm-" + uuid.uuid4().hex
    chunks: list[bytes] = []
    for name, value in (fields or {}).items():
        chunks.append(f"--{boundary}\r\n".encode())
        chunks.append(f'Content-Disposition: form-data; name="{name}"\r\n\r\n'.encode())
        chunks.append(str(value).encode())
        chunks.append(b"\r\n")
    content_type = mimetypes.guess_type(path.name)[0] or "application/octet-stream"
    chunks.append(f"--{boundary}\r\n".encode())
    chunks.append(f'Content-Disposition: form-data; name="{field}"; filename="{path.name}"\r\n'.encode())
    chunks.append(f"Content-Type: {content_type}\r\n\r\n".encode())
    chunks.append(path.read_bytes())
    chunks.append(b"\r\n")
    chunks.append(f"--{boundary}--\r\n".encode())
    req = urllib.request.Request(url, data=b"".join(chunks), method="POST")
    req.add_header("Content-Type", f"multipart/form-data; boundary={boundary}")
    for key, value in (headers or {}).items():
        req.add_header(key, value)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:  # noqa: S310 - configured service
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8", "replace")
        raise RuntimeError(f"HTTP {e.code}: {body[:400]}") from e


def citadel_headers(api_key: str) -> dict[str, str]:
    if not api_key:
        return {}
    return {"X-API-Key": api_key, "Authorization": f"Bearer {api_key}", "X-Internal-Token": api_key}


def load_dotenv(path: str = ".env") -> None:
    env_path = Path(path)
    if not env_path.exists():
        return
    for raw in env_path.read_text().splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        value = value.split("#", 1)[0].strip().strip('"').strip("'")
        if key.strip() and key.strip() not in os.environ:
            os.environ[key.strip()] = value


if __name__ == "__main__":
    raise SystemExit(main())
