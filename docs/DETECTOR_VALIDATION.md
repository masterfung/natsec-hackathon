# Detector Validation

Generated: 2026-05-03T05:14:56Z

## Sample Inventory

| file | source | label | cosine vs real_01 | Citadel deepfake | notes |
|---|---:|---:|---:|---:|---|
| `clone_cartesia_01.wav` | cartesia | synthetic | 0.816395 | 0 |  |
| `clone_cartesia_02.wav` | cartesia | synthetic | 0.817039 | 0 |  |
| `clone_cartesia_03.wav` | cartesia | synthetic | 0.812518 | 0 |  |
| `clone_cartesia_04.wav` | cartesia | synthetic | 0.774359 | 0 |  |
| `clone_cartesia_05.wav` | cartesia | synthetic | 0.780077 | 0 |  |
| `clone_cartesia_06.wav` | cartesia | synthetic | 0.834483 | 0 |  |
| `clone_cartesia_07.wav` | cartesia | synthetic | 0.802081 | 0 |  |
| `clone_cartesia_08.wav` | cartesia | synthetic | 0.790790 | 0 |  |
| `clone_cartesia_09.wav` | cartesia | synthetic | 0.797729 | 0 |  |
| `clone_cartesia_10.wav` | cartesia | synthetic | 0.752965 | 45 |  |
| `clone_cartesia_11.wav` | cartesia | synthetic | 0.799068 | 0 |  |
| `clone_cartesia_12.wav` | cartesia | synthetic | 0.801685 | 0 |  |
| `clone_cartesia_13.wav` | cartesia | synthetic | 0.821798 | 0 |  |
| `clone_cartesia_14.wav` | cartesia | synthetic | 0.817990 | 0 |  |
| `clone_cartesia_15.wav` | cartesia | synthetic | 0.814330 | 0 |  |
| `clone_cartesia_16.wav` | cartesia | synthetic | 0.792350 | 0 |  |
| `clone_cartesia_17.wav` | cartesia | synthetic | 0.801019 | 0 |  |
| `clone_cartesia_18.wav` | cartesia | synthetic | 0.805600 | 0 |  |
| `clone_cartesia_19.wav` | cartesia | synthetic | 0.826408 | 0 |  |
| `clone_cartesia_20.wav` | cartesia | synthetic | 0.733482 | 0 |  |
| `clone_cartesia_21.wav` | cartesia | synthetic | 0.775565 | 0 |  |
| `clone_cartesia_22.wav` | cartesia | synthetic | 0.819386 | 0 |  |
| `clone_cartesia_23.wav` | cartesia | synthetic | 0.784744 | 0 |  |
| `clone_cartesia_24.wav` | cartesia | synthetic | 0.775195 | 0 |  |
| `real_01.wav` | real | real | 1.000000 | 0 |  |
| `real_02.wav` | real | real | 0.872901 | 0 |  |
| `real_03.wav` | real | real | 0.905535 | 0 |  |
| `real_04.wav` | real | real | 0.831734 | 0 |  |
| `real_05.wav` | real | real | 0.558101 | 0 |  |
| `real_06.wav` | real | real | 0.699189 | 0 |  |
| `real_07.wav` | real | real | 0.760848 | 0 |  |
| `real_08.wav` | real | real | 0.781890 | 51 |  |

## Distributions

- Real deepfake risk: n=8, mean=6.38, p50=0.00, p95=33.15
- Clone deepfake risk: n=24, mean=1.88, p50=0.00, p95=0.00
- Real overall Citadel risk: n=8, mean=61.75, p50=65.00, p95=81.90
- Clone overall Citadel risk: n=24, mean=54.62, p50=62.50, p95=94.85
- Real cosine vs real_01: n=8, mean=0.80, p50=0.81, p95=0.97
- Clone/control cosine vs real_01: n=24, mean=0.80, p50=0.80, p95=0.83

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

FAIL for passive deepfake separation: measured values exist, but clone scores are not higher than real scores under the plan rule. Reframe the demo around OOB as the decisive factor.

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
- mode: `secure`
