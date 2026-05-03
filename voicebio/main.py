"""Mighty Morphing — Voice Biometric Sidecar

FastAPI service exposing speaker-embedding extraction and matching using
SpeechBrain ECAPA-TDNN. Called by the Go gateway for /api/enroll and /api/auth.

Endpoints:
- GET  /health       liveness + model state
- POST /embed        audio file in -> 192-dim embedding out
- POST /match        two embeddings -> cosine similarity + threshold check
"""

from __future__ import annotations

import io
import logging
import os
import subprocess
import time
from contextlib import asynccontextmanager

import numpy as np
import soundfile as sf
import torch
import torchaudio
from fastapi import FastAPI, HTTPException, UploadFile
from pydantic import BaseModel, Field

logger = logging.getLogger("voicebio")
logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")

EMBED_DIM = 192
SAMPLE_RATE = 16000
DEFAULT_MATCH_THRESHOLD = float(os.getenv("MM_SPEAKER_MATCH_THRESHOLD", "0.75"))
MIN_AUDIO_SECONDS = 1.0
MODEL_CACHE_DIR = os.getenv("VOICEBIO_MODEL_CACHE", "/tmp/speechbrain-cache")

_classifier = None
_loaded_at: float | None = None


def get_classifier():
    """Lazy-load the ECAPA-TDNN model. ~80MB, downloads on first call."""
    global _classifier, _loaded_at
    if _classifier is None:
        t0 = time.time()
        logger.info("loading SpeechBrain ECAPA-TDNN (this can take ~10s on first run)...")
        from speechbrain.inference.speaker import EncoderClassifier  # noqa: PLC0415

        _classifier = EncoderClassifier.from_hparams(
            source="speechbrain/spkrec-ecapa-voxceleb",
            savedir=MODEL_CACHE_DIR,
            run_opts={"device": "cpu"},
        )
        _loaded_at = time.time()
        logger.info("model loaded in %.2fs", _loaded_at - t0)
    return _classifier


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Pre-warm the model at startup so first /embed isn't slow.
    try:
        get_classifier()
    except Exception as e:  # noqa: BLE001
        logger.error("failed to pre-warm model: %s", e)
    yield


app = FastAPI(
    title="Mighty Morphing — Voice Biometric",
    version="0.1.0",
    lifespan=lifespan,
)


class HealthResponse(BaseModel):
    status: str
    model_loaded: bool
    embed_dim: int
    sample_rate: int
    loaded_at_unix: float | None = None


@app.get("/health", response_model=HealthResponse)
def health() -> HealthResponse:
    return HealthResponse(
        status="ok",
        model_loaded=_classifier is not None,
        embed_dim=EMBED_DIM,
        sample_rate=SAMPLE_RATE,
        loaded_at_unix=_loaded_at,
    )


class EmbedResponse(BaseModel):
    embedding: list[float] = Field(..., description="192-dim float32 ECAPA-TDNN embedding")
    dim: int
    duration_s: float
    quality: float = Field(..., description="Heuristic quality score 0..1 (proxy for SNR/level)")
    inference_ms: int


def _resample_if_needed(waveform: torch.Tensor, sr: int) -> tuple[torch.Tensor, int]:
    if sr != SAMPLE_RATE:
        resampler = torchaudio.transforms.Resample(sr, SAMPLE_RATE)
        waveform = resampler(waveform)
        sr = SAMPLE_RATE
    return waveform, sr


def _decode_with_soundfile(raw: bytes) -> tuple[torch.Tensor, int]:
    audio_np, sr = sf.read(io.BytesIO(raw), dtype="float32", always_2d=False)
    if audio_np.ndim == 2:
        audio_np = audio_np.mean(axis=1)
    waveform = torch.from_numpy(audio_np).unsqueeze(0)  # [1, samples]
    return _resample_if_needed(waveform, int(sr))


def _decode_with_ffmpeg(raw: bytes) -> tuple[torch.Tensor, int]:
    """Decode browser containers like WebM/Opus through ffmpeg to mono f32 PCM."""
    cmd = [
        "ffmpeg",
        "-hide_banner",
        "-loglevel",
        "error",
        "-nostdin",
        "-i",
        "pipe:0",
        "-f",
        "f32le",
        "-acodec",
        "pcm_f32le",
        "-ac",
        "1",
        "-ar",
        str(SAMPLE_RATE),
        "pipe:1",
    ]
    try:
        proc = subprocess.run(cmd, input=raw, capture_output=True, check=False, timeout=15)
    except FileNotFoundError as e:
        raise RuntimeError("ffmpeg is required to decode this audio container") from e
    except subprocess.TimeoutExpired as e:
        raise RuntimeError("ffmpeg audio decode timed out") from e
    if proc.returncode != 0:
        detail = proc.stderr.decode("utf-8", errors="replace").strip()
        raise RuntimeError(f"ffmpeg audio decode failed: {detail or 'unknown error'}")
    audio_np = np.frombuffer(proc.stdout, dtype=np.float32)
    if audio_np.size == 0:
        raise RuntimeError("ffmpeg decoded no audio samples")
    waveform = torch.from_numpy(audio_np.copy()).unsqueeze(0)  # [1, samples]
    return waveform, SAMPLE_RATE


def _decode_audio(raw: bytes) -> tuple[torch.Tensor, int]:
    """Decode uploaded audio to mono float32 tensor at 16kHz."""
    try:
        return _decode_with_soundfile(raw)
    except Exception as soundfile_err:  # noqa: BLE001
        try:
            return _decode_with_ffmpeg(raw)
        except Exception as ffmpeg_err:  # noqa: BLE001
            raise RuntimeError(f"soundfile: {soundfile_err}; ffmpeg: {ffmpeg_err}") from ffmpeg_err


@app.post("/embed", response_model=EmbedResponse)
async def embed(audio: UploadFile) -> EmbedResponse:
    raw = await audio.read()
    if not raw:
        raise HTTPException(400, "empty audio body")

    try:
        waveform, sr = _decode_audio(raw)
    except Exception as e:  # noqa: BLE001
        raise HTTPException(400, f"audio decode failed: {e}") from e

    duration = waveform.shape[1] / sr
    if duration < MIN_AUDIO_SECONDS:
        raise HTTPException(400, f"audio too short ({duration:.2f}s); need >= {MIN_AUDIO_SECONDS}s")

    classifier = get_classifier()
    t0 = time.time()
    with torch.no_grad():
        embedding = classifier.encode_batch(waveform).squeeze().cpu().numpy().astype(np.float32)
    inference_ms = int((time.time() - t0) * 1000)

    if embedding.shape[0] != EMBED_DIM:
        raise HTTPException(500, f"unexpected embedding dim {embedding.shape[0]}, expected {EMBED_DIM}")

    quality = float(min(1.0, waveform.abs().mean().item() * 10.0))

    return EmbedResponse(
        embedding=embedding.tolist(),
        dim=int(embedding.shape[0]),
        duration_s=duration,
        quality=quality,
        inference_ms=inference_ms,
    )


class MatchRequest(BaseModel):
    enrolled: list[float] = Field(..., min_length=EMBED_DIM, max_length=EMBED_DIM)
    candidate: list[float] = Field(..., min_length=EMBED_DIM, max_length=EMBED_DIM)
    threshold: float | None = Field(default=None, description="Override default threshold")


class MatchResponse(BaseModel):
    similarity: float
    threshold: float
    is_match: bool


@app.post("/match", response_model=MatchResponse)
def match(req: MatchRequest) -> MatchResponse:
    a = np.asarray(req.enrolled, dtype=np.float32)
    b = np.asarray(req.candidate, dtype=np.float32)
    norm = float(np.linalg.norm(a) * np.linalg.norm(b))
    if norm < 1e-9:
        raise HTTPException(400, "zero-norm embedding")
    similarity = float(np.dot(a, b) / norm)
    threshold = req.threshold if req.threshold is not None else DEFAULT_MATCH_THRESHOLD
    return MatchResponse(
        similarity=similarity,
        threshold=threshold,
        is_match=similarity >= threshold,
    )
