# Detector Validation

Generated: 2026-05-03T04:36:26Z

## Sample Inventory

| file | source | label | cosine vs real_01 | Citadel deepfake | notes |
|---|---:|---:|---:|---:|---|
| `clone_cartesia_01.wav` | cartesia | synthetic | 0.804868 | - |  |
| `clone_cartesia_02.wav` | cartesia | synthetic | 0.777119 | - |  |
| `clone_cartesia_03.wav` | cartesia | synthetic | 0.827843 | - |  |
| `real_01.wav` | real | real | 1.000000 | - |  |
| `real_02.wav` | real | real | 0.872901 | - |  |
| `real_03.wav` | real | real | 0.905535 | - |  |
| `real_04.wav` | real | real | 0.831734 | - |  |
| `real_05.wav` | real | real | 0.558101 | - |  |
| `real_06.wav` | real | real | 0.699189 | - |  |
| `real_07.wav` | real | real | 0.760848 | - |  |
| `real_08.wav` | real | real | 0.781890 | 51 |  |

## Distributions

- Real deepfake risk: n=8, mean=6.38, p50=0.00, p95=33.15
- Clone deepfake risk: n=3, mean=0.00, p50=0.00, p95=0.00
- Real cosine vs real_01: n=8, mean=0.80, p50=0.81, p95=0.97
- Clone/control cosine vs real_01: n=3, mean=0.80, p50=0.80, p95=0.83

## Speaker Baseline Analysis

| file | cosine vs real_01 | cosine vs real centroid |
|---|---:|---:|
| `real_01.wav` | 1.000 | 0.923 |
| `real_02.wav` | 0.873 | 0.923 |
| `real_03.wav` | 0.906 | 0.904 |
| `real_04.wav` | 0.832 | 0.926 |
| `real_05.wav` | 0.558 | 0.697 |
| `real_06.wav` | 0.699 | 0.808 |
| `real_07.wav` | 0.761 | 0.879 |
| `real_08.wav` | 0.782 | 0.871 |

Pairwise real-vs-real cosine: n=28, mean=0.72, p50=0.76, p95=0.87

Interpretation: single-anchor matching is sensitive to modulation and recording condition. A production enrollment should store multiple templates or a centroid plus outlier policy, not one embedding.

## Verdict

PARTIAL PASS: measured values exist, but they do not satisfy the full separation rule. Use OOB as the decisive demo factor and document thresholds honestly.

## Threshold Recommendation

- Deepfake threshold not separable at 95/5; keep conservative default and rely on OOB.
- `MM_SPEAKER_MATCH_THRESHOLD=0.59` from real-sample p05 minus margin.

## Reproduction

```bash
cd natsec
python3 scripts/clone_voice.py
python3 scripts/validate_detector.py
```

## Configuration

- voicebio: `http://localhost:7100`
- citadel_audio: `https://your-citadel-audio.example`
- mode: `comprehensive`
