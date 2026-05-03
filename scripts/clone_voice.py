#!/usr/bin/env python3
"""Generate clone fixtures for Phase 0 detector validation.

Inputs:
  fixtures/user/real_01.wav (or .mp3/.m4a/.flac)

Outputs:
  fixtures/user/clone_cartesia_*.wav
  fixtures/user/clone_elevenlabs_*.mp3

Provider API notes verified against official docs on 2026-05-02:
  - Cartesia POST /voices/clone and POST /tts/bytes
  - ElevenLabs POST /v1/voices/add and POST /v1/text-to-speech/{voice_id}
"""

from __future__ import annotations

import argparse
import json
import mimetypes
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid
from pathlib import Path


DEFAULT_UTTERANCES = [
    "PACOM, this is the duty officer. Confirming receipt of the situation report at zero nine hundred local. Acknowledged, out.",
    "Route the latest logistics summary to the command channel and hold execution until identity verification clears.",
    "This is a controlled detector validation sample for Mighty Morphing. The words are synthetic and should be blocked by the gateway.",
]


def main() -> int:
    load_dotenv()
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--fixtures-dir", default="fixtures/user", help="Directory containing real_01 and receiving clone_* outputs")
    parser.add_argument("--utterance", action="append", help="Text to synthesize; may be passed multiple times")
    parser.add_argument("--skip-cartesia", action="store_true")
    parser.add_argument("--skip-elevenlabs", action="store_true")
    args = parser.parse_args()

    fixtures_dir = Path(args.fixtures_dir)
    fixtures_dir.mkdir(parents=True, exist_ok=True)
    source = first_existing(fixtures_dir, "real_01", [".wav", ".mp3", ".m4a", ".flac", ".ogg", ".aiff"])
    if source is None:
        print(f"missing {fixtures_dir}/real_01.*; record the Phase 0 enrollment sample first", file=sys.stderr)
        return 2

    utterances = args.utterance or DEFAULT_UTTERANCES
    manifest: dict[str, object] = {"source": str(source), "created_at": utc_now(), "outputs": []}

    if not args.skip_cartesia:
        try:
            cartesia_outputs = run_cartesia(source, fixtures_dir, utterances)
            manifest["outputs"].extend(cartesia_outputs)
        except ProviderSkipped as e:
            print(f"cartesia skipped: {e}", file=sys.stderr)
        except Exception as e:  # noqa: BLE001
            print(f"cartesia failed: {e}", file=sys.stderr)

    if not args.skip_elevenlabs:
        try:
            eleven_outputs = run_elevenlabs(source, fixtures_dir, utterances)
            manifest["outputs"].extend(eleven_outputs)
        except ProviderSkipped as e:
            print(f"elevenlabs skipped: {e}", file=sys.stderr)
        except Exception as e:  # noqa: BLE001
            print(f"elevenlabs failed: {e}", file=sys.stderr)

    manifest_path = fixtures_dir / "clone_manifest.json"
    manifest_path.write_text(json.dumps(manifest, indent=2) + "\n")
    print(f"wrote {manifest_path}")
    return 0


class ProviderSkipped(RuntimeError):
    pass


def run_cartesia(source: Path, fixtures_dir: Path, utterances: list[str]) -> list[dict[str, str]]:
    key = os.getenv("CARTESIA_API_KEY", "").strip()
    if not key:
        raise ProviderSkipped("CARTESIA_API_KEY not set")
    version = os.getenv("CARTESIA_VERSION", "2026-03-01")
    model_id = os.getenv("CARTESIA_MODEL_ID", "sonic-3")
    api_base = os.getenv("CARTESIA_API_BASE", "https://api.cartesia.ai").rstrip("/")

    voice_name = f"mm-phase0-{int(time.time())}"
    clone = multipart_request(
        f"{api_base}/voices/clone",
        headers={"X-API-Key": key, "Cartesia-Version": version},
        fields={
            "name": voice_name,
            "description": "Mighty Morphing Phase 0 detector validation clone",
            "language": "en",
            "mode": os.getenv("CARTESIA_CLONE_MODE", "similarity"),
            "enhance": os.getenv("CARTESIA_ENHANCE", "true"),
        },
        files={"clip": source},
    )
    clone_body = json.loads(clone.decode("utf-8"))
    voice_id = clone_body["id"]
    outputs: list[dict[str, str]] = []
    for idx, text in enumerate(utterances, 1):
        payload = {
            "model_id": model_id,
            "transcript": text,
            "voice": {"mode": "id", "id": voice_id},
            "output_format": {"container": "wav", "encoding": "pcm_s16le", "sample_rate": 16000},
            "language": "en",
        }
        audio = json_request(
            f"{api_base}/tts/bytes",
            headers={"X-API-Key": key, "Cartesia-Version": version},
            payload=payload,
            expect_json=False,
        )
        out = fixtures_dir / f"clone_cartesia_{idx:02d}.wav"
        out.write_bytes(audio)
        outputs.append({"provider": "cartesia", "path": str(out), "voice_id": voice_id})
        print(f"wrote {out}")
    return outputs


def run_elevenlabs(source: Path, fixtures_dir: Path, utterances: list[str]) -> list[dict[str, str]]:
    key = os.getenv("ELEVENLABS_API_KEY", "").strip()
    if not key:
        raise ProviderSkipped("ELEVENLABS_API_KEY not set")
    api_base = os.getenv("ELEVENLABS_API_BASE", "https://api.elevenlabs.io").rstrip("/")
    model_id = os.getenv("ELEVENLABS_MODEL_ID", "eleven_multilingual_v2")

    create = multipart_request(
        f"{api_base}/v1/voices/add",
        headers={"xi-api-key": key},
        fields={
            "name": f"mm-phase0-{int(time.time())}",
            "description": "Mighty Morphing Phase 0 detector validation clone",
            "remove_background_noise": "false",
        },
        files={"files[]": source},
    )
    create_body = json.loads(create.decode("utf-8"))
    voice_id = create_body["voice_id"]
    outputs: list[dict[str, str]] = []
    for idx, text in enumerate(utterances, 1):
        query = urllib.parse.urlencode({"output_format": os.getenv("ELEVENLABS_OUTPUT_FORMAT", "mp3_44100_128")})
        audio = json_request(
            f"{api_base}/v1/text-to-speech/{voice_id}?{query}",
            headers={"xi-api-key": key},
            payload={
                "text": text,
                "model_id": model_id,
                "voice_settings": {"stability": 0.45, "similarity_boost": 0.9},
            },
            expect_json=False,
        )
        out = fixtures_dir / f"clone_elevenlabs_{idx:02d}.mp3"
        out.write_bytes(audio)
        outputs.append({"provider": "elevenlabs", "path": str(out), "voice_id": voice_id})
        print(f"wrote {out}")
    return outputs


def json_request(url: str, headers: dict[str, str], payload: object, expect_json: bool = True) -> bytes:
    data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(url, data=data, method="POST")
    req.add_header("Content-Type", "application/json")
    for key, value in headers.items():
        req.add_header(key, value)
    return open_request(req, expect_json=expect_json)


def multipart_request(url: str, headers: dict[str, str], fields: dict[str, str], files: dict[str, Path]) -> bytes:
    boundary = "----mm-" + uuid.uuid4().hex
    chunks: list[bytes] = []
    for name, value in fields.items():
        chunks.append(f"--{boundary}\r\n".encode())
        chunks.append(f'Content-Disposition: form-data; name="{name}"\r\n\r\n'.encode())
        chunks.append(str(value).encode())
        chunks.append(b"\r\n")
    for name, path in files.items():
        filename = path.name
        content_type = mimetypes.guess_type(filename)[0] or "application/octet-stream"
        chunks.append(f"--{boundary}\r\n".encode())
        chunks.append(f'Content-Disposition: form-data; name="{name}"; filename="{filename}"\r\n'.encode())
        chunks.append(f"Content-Type: {content_type}\r\n\r\n".encode())
        chunks.append(path.read_bytes())
        chunks.append(b"\r\n")
    chunks.append(f"--{boundary}--\r\n".encode())
    req = urllib.request.Request(url, data=b"".join(chunks), method="POST")
    req.add_header("Content-Type", f"multipart/form-data; boundary={boundary}")
    for key, value in headers.items():
        req.add_header(key, value)
    return open_request(req, expect_json=True)


def open_request(req: urllib.request.Request, expect_json: bool) -> bytes:
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:  # noqa: S310 - explicit user API targets
            body = resp.read()
            if expect_json:
                json.loads(body.decode("utf-8"))
            return body
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8", "replace")
        raise RuntimeError(f"HTTP {e.code}: {body[:600]}") from e


def first_existing(base: Path, stem: str, suffixes: list[str]) -> Path | None:
    for suffix in suffixes:
        path = base / f"{stem}{suffix}"
        if path.exists():
            return path
    return None


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
