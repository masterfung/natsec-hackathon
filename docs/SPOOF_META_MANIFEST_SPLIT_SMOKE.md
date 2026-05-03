# Spoof Meta-Detector Experiment

Samples: 32 (8 real, 24 synthetic)

## Holdout Result

- Evaluation: stratified holdout split, test_size=0.25, seed=7.
- Evaluated samples: 8/32.
- best_zero_fp=thr=0.973, TPR=1.00, FPR=0.00, TP=6, FP=0, TN=2, FN=0; best_balanced=thr=0.973, TPR=1.00, FPR=0.00, TP=6, FP=0, TN=2, FN=0

## Scores

| file | label | split | holdout spoof score | in-sample score | current deepfake | current overall | latency ms |
|---|---:|---:|---:|---:|---:|---:|---:|
| `clone_cartesia_23.wav` | synthetic | eval | 1.000 | 1.000 |  |  |  |
| `clone_cartesia_15.wav` | synthetic | eval | 0.999 | 0.999 |  |  |  |
| `clone_cartesia_13.wav` | synthetic | eval | 0.999 | 0.999 |  |  |  |
| `clone_cartesia_05.wav` | synthetic | eval | 0.997 | 0.996 |  |  |  |
| `clone_cartesia_12.wav` | synthetic | eval | 0.987 | 0.984 |  |  |  |
| `clone_cartesia_10.wav` | synthetic | eval | 0.973 | 0.978 |  |  |  |
| `real_01.wav` | real | eval | 0.749 | 0.303 |  |  |  |
| `real_07.wav` | real | eval | 0.002 | 0.001 |  |  |  |
| `clone_cartesia_01.wav` | synthetic | train |  | 0.931 |  |  |  |
| `clone_cartesia_02.wav` | synthetic | train |  | 0.999 |  |  |  |
| `clone_cartesia_03.wav` | synthetic | train |  | 1.000 |  |  |  |
| `clone_cartesia_04.wav` | synthetic | train |  | 0.974 |  |  |  |
| `clone_cartesia_06.wav` | synthetic | train |  | 0.998 |  |  |  |
| `clone_cartesia_07.wav` | synthetic | train |  | 0.962 |  |  |  |
| `clone_cartesia_08.wav` | synthetic | train |  | 0.994 |  |  |  |
| `clone_cartesia_09.wav` | synthetic | train |  | 0.989 |  |  |  |
| `clone_cartesia_11.wav` | synthetic | train |  | 0.998 |  |  |  |
| `clone_cartesia_14.wav` | synthetic | train |  | 0.933 |  |  |  |
| `clone_cartesia_16.wav` | synthetic | train |  | 0.996 |  |  |  |
| `clone_cartesia_17.wav` | synthetic | train |  | 0.982 |  |  |  |
| `clone_cartesia_18.wav` | synthetic | train |  | 0.959 |  |  |  |
| `clone_cartesia_19.wav` | synthetic | train |  | 0.998 |  |  |  |
| `clone_cartesia_20.wav` | synthetic | train |  | 1.000 |  |  |  |
| `clone_cartesia_21.wav` | synthetic | train |  | 0.972 |  |  |  |
| `clone_cartesia_22.wav` | synthetic | train |  | 0.930 |  |  |  |
| `clone_cartesia_24.wav` | synthetic | train |  | 0.993 |  |  |  |
| `real_02.wav` | real | train |  | 0.004 |  |  |  |
| `real_03.wav` | real | train |  | 0.004 |  |  |  |
| `real_04.wav` | real | train |  | 0.005 |  |  |  |
| `real_05.wav` | real | train |  | 0.002 |  |  |  |
| `real_06.wav` | real | train |  | 0.000 |  |  |  |
| `real_08.wav` | real | train |  | 0.081 |  |  |  |

## Operating Bands

- Scores are calibrated on held-out predictions, not in-sample predictions.
- Max real score: 0.749; min synthetic score: 0.973; nearest detected synthetic above the real range: 0.973.
- Suggested legacy WARN threshold: 0.811. Suggested legacy BLOCK threshold: 0.861.
- At BLOCK threshold: thr=0.861, TPR=1.00, FPR=0.00, TP=6, FP=0, TN=2, FN=0.
- At catch-all synthetic threshold: thr=0.973, TPR=1.00, FPR=0.00, TP=6, FP=0, TN=2, FN=0.

## Feature-Set Ablations

| feature set | zero-FP result | max real | min synthetic | suggested block | catch-all result |
|---|---|---:|---:|---:|---|
| `all` | thr=0.973, TPR=1.00, FPR=0.00, TP=6, FP=0, TN=2, FN=0 | 0.749 | 0.973 | 0.861 | thr=0.973, TPR=1.00, FPR=0.00, TP=6, FP=0, TN=2, FN=0 |
| `local+speaker_fast` | thr=0.973, TPR=1.00, FPR=0.00, TP=6, FP=0, TN=2, FN=0 | 0.749 | 0.973 | 0.861 | thr=0.973, TPR=1.00, FPR=0.00, TP=6, FP=0, TN=2, FN=0 |
| `local_only_fast` | thr=0.973, TPR=1.00, FPR=0.00, TP=6, FP=0, TN=2, FN=0 | 0.749 | 0.973 | 0.861 | thr=0.973, TPR=1.00, FPR=0.00, TP=6, FP=0, TN=2, FN=0 |
| `citadel_only` | none | 0.750 | 0.750 | 0.750 | thr=0.750, TPR=1.00, FPR=1.00, TP=6, FP=2, TN=0, FN=0 |
| `speaker_only` | none | 0.750 | 0.750 | 0.750 | thr=0.750, TPR=1.00, FPR=1.00, TP=6, FP=2, TN=0, FN=0 |

## Group Holdouts

| held-out column | mode | groups | covered samples | skipped groups | zero-FP result |
|---|---|---:|---:|---:|---|
| `dataset` | split | 0 | 0 | 0 | need at least two groups |
| `generator_id` | split | 0 | 0 | 0 | could not find a group split with both classes in train and test |
| `speaker_id` | split | 0 | 0 | 0 | need at least two groups |

## Current-Signal Baselines

- `sig_deepfake_detection` best balanced threshold: thr=0.000, TPR=1.00, FPR=1.00, TP=24, FP=8, TN=0, FN=0
- `citadel_risk` best balanced threshold: thr=0.000, TPR=1.00, FPR=1.00, TP=24, FP=8, TN=0, FN=0
- `sig_wavlm_paralinguistic` best balanced threshold: thr=0.000, TPR=1.00, FPR=1.00, TP=24, FP=8, TN=0, FN=0
- `sig_rmt_divergence` best balanced threshold: thr=0.000, TPR=1.00, FPR=1.00, TP=24, FP=8, TN=0, FN=0

## Largest Coefficients

| feature | weight |
|---|---:|
| `bias` | 2.558 |
| `flatness_std` | -1.489 |
| `pitch_cv` | -1.298 |
| `pitch_std` | -1.067 |
| `rms_std` | -1.019 |
| `silence_ratio` | -0.856 |
| `rms_mean` | -0.537 |
| `pitch_mean` | -0.337 |
| `centroid_std` | -0.267 |
| `centroid_mean` | -0.091 |
| `flatness_mean` | -0.059 |
| `hf_ratio_mean` | -0.055 |

## Interpretation

This experiment can show whether the available features contain signal, but it does not prove generalization. A deployable detector needs more real speakers, more clone providers, and provider-held-out evaluation.

## Model Artifact

```json
{
  "features": [
    "speaker_cosine_vs_real_01",
    "citadel_risk",
    "sig_spectral_anomaly",
    "sig_steganalysis",
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
    "pitch_cv"
  ],
  "weights": [
    2.558239100263374,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    -0.5368418789619791,
    -1.0194520720475873,
    -0.8563892868636699,
    -0.09149616312906207,
    -0.2667963525064571,
    -0.05928678195703628,
    -1.4893152257550448,
    -0.05525120859435143,
    0.04820987044490514,
    -0.33652117097941175,
    -1.0672024634197164,
    -1.2976606670513144
  ],
  "mean": [
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0,
    0.0589818125,
    0.06440759374999999,
    0.28567765625,
    1134.2742187500003,
    1313.1450625,
    0.21303246875,
    0.16299762499999998,
    0.09247275,
    0.15358171874999998,
    140.5805625,
    63.69521875,
    0.44739249999999997
  ],
  "std": [
    1.0,
    1.0,
    1.0,
    1.0,
    1.0,
    1.0,
    1.0,
    1.0,
    1.0,
    0.007415972746534588,
    0.006834749468979163,
    0.06332325701737147,
    177.82544841865547,
    204.25285400544857,
    0.039057735225388114,
    0.016143438902829068,
    0.03261728702984815,
    0.022425770871971125,
    13.827263261708506,
    18.05973661341434,
    0.08946199113345568
  ],
  "evaluation": "stratified holdout split, test_size=0.25, seed=7."
}
```
