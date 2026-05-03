# Spoof Meta-Detector Experiment

Samples: 32 (8 real, 24 synthetic)

## Holdout Result

- Evaluation: leave-one-out. This is still small-sample and provider-specific.
- best_zero_fp=thr=0.867, TPR=0.96, FPR=0.00, TP=23, FP=0, TN=8, FN=1; best_balanced=thr=0.867, TPR=0.96, FPR=0.00, TP=23, FP=0, TN=8, FN=1

## Scores

| file | label | LOO spoof score | in-sample score | current deepfake | current overall | latency ms |
|---|---:|---:|---:|---:|---:|---:|
| `clone_cartesia_03.wav` | synthetic | 1.000 | 1.000 | 0 | 15 | 651 |
| `clone_cartesia_20.wav` | synthetic | 1.000 | 1.000 | 0 | 65 | 478 |
| `clone_cartesia_23.wav` | synthetic | 1.000 | 1.000 | 0 | 65 | 525 |
| `clone_cartesia_02.wav` | synthetic | 0.999 | 0.999 | 0 | 65 | 567 |
| `clone_cartesia_15.wav` | synthetic | 0.999 | 0.999 | 0 | 90 | 280 |
| `clone_cartesia_05.wav` | synthetic | 0.999 | 0.999 | 0 | 15 | 595 |
| `clone_cartesia_06.wav` | synthetic | 0.999 | 0.998 | 0 | 15 | 574 |
| `clone_cartesia_08.wav` | synthetic | 0.998 | 0.998 | 0 | 15 | 498 |
| `clone_cartesia_19.wav` | synthetic | 0.998 | 0.998 | 0 | 39 | 559 |
| `clone_cartesia_24.wav` | synthetic | 0.997 | 0.997 | 0 | 39 | 494 |
| `clone_cartesia_13.wav` | synthetic | 0.997 | 0.997 | 0 | 60 | 525 |
| `clone_cartesia_11.wav` | synthetic | 0.996 | 0.996 | 0 | 65 | 568 |
| `clone_cartesia_09.wav` | synthetic | 0.992 | 0.993 | 0 | 39 | 536 |
| `clone_cartesia_04.wav` | synthetic | 0.990 | 0.992 | 0 | 17 | 585 |
| `clone_cartesia_16.wav` | synthetic | 0.990 | 0.992 | 0 | 95 | 565 |
| `clone_cartesia_21.wav` | synthetic | 0.984 | 0.987 | 0 | 94 | 254 |
| `clone_cartesia_12.wav` | synthetic | 0.978 | 0.983 | 0 | 65 | 523 |
| `clone_cartesia_18.wav` | synthetic | 0.966 | 0.984 | 0 | 92 | 261 |
| `clone_cartesia_01.wav` | synthetic | 0.953 | 0.974 | 0 | 60 | 842 |
| `clone_cartesia_14.wav` | synthetic | 0.929 | 0.973 | 0 | 15 | 582 |
| `clone_cartesia_10.wav` | synthetic | 0.921 | 0.981 | 45 | 98 | 398 |
| `clone_cartesia_17.wav` | synthetic | 0.890 | 0.970 | 0 | 84 | 602 |
| `clone_cartesia_22.wav` | synthetic | 0.867 | 0.948 | 0 | 39 | 500 |
| `real_01.wav` | real | 0.834 | 0.097 | 0 | 65 | 856 |
| `clone_cartesia_07.wav` | synthetic | 0.820 | 0.967 | 0 | 39 | 582 |
| `real_08.wav` | real | 0.265 | 0.028 | 51 | 39 | 10848 |
| `real_04.wav` | real | 0.070 | 0.031 | 0 | 91 | 254 |
| `real_05.wav` | real | 0.039 | 0.011 | 0 | 65 | 576 |
| `real_03.wav` | real | 0.014 | 0.010 | 0 | 65 | 714 |
| `real_07.wav` | real | 0.013 | 0.009 | 0 | 65 | 635 |
| `real_06.wav` | real | 0.007 | 0.006 | 0 | 39 | 461 |
| `real_02.wav` | real | 0.002 | 0.002 | 0 | 65 | 573 |

## Operating Bands

- Scores are calibrated on leave-one-out predictions, not in-sample predictions.
- Max real score: 0.834; min synthetic score: 0.820; nearest detected synthetic above the real range: 0.867.
- Suggested legacy WARN threshold: 0.800. Suggested legacy BLOCK threshold: 0.850.
- At BLOCK threshold: thr=0.850, TPR=0.96, FPR=0.00, TP=23, FP=0, TN=8, FN=1.
- At catch-all synthetic threshold: thr=0.820, TPR=1.00, FPR=0.12, TP=24, FP=1, TN=7, FN=0.

## Feature-Set Ablations

| feature set | zero-FP result | max real | min synthetic | suggested block | catch-all result |
|---|---|---:|---:|---:|---|
| `all` | thr=0.867, TPR=0.96, FPR=0.00, TP=23, FP=0, TN=8, FN=1 | 0.834 | 0.820 | 0.850 | thr=0.820, TPR=1.00, FPR=0.12, TP=24, FP=1, TN=7, FN=0 |
| `local+speaker_fast` | thr=0.912, TPR=0.96, FPR=0.00, TP=23, FP=0, TN=8, FN=1 | 0.747 | 0.510 | 0.830 | thr=0.510, TPR=1.00, FPR=0.12, TP=24, FP=1, TN=7, FN=0 |
| `local_only_fast` | thr=0.813, TPR=1.00, FPR=0.00, TP=24, FP=0, TN=8, FN=0 | 0.735 | 0.813 | 0.774 | thr=0.813, TPR=1.00, FPR=0.00, TP=24, FP=0, TN=8, FN=0 |
| `citadel_only` | thr=0.997, TPR=0.12, FPR=0.00, TP=3, FP=0, TN=8, FN=21 | 0.990 | 0.028 | 0.994 | thr=0.028, TPR=1.00, FPR=0.88, TP=24, FP=7, TN=1, FN=0 |
| `speaker_only` | none | 0.990 | 0.734 | 0.990 | thr=0.734, TPR=1.00, FPR=1.00, TP=24, FP=8, TN=0, FN=0 |

## Current-Signal Baselines

- `sig_deepfake_detection` best balanced threshold: thr=0.000, TPR=1.00, FPR=1.00, TP=24, FP=8, TN=0, FN=0
- `citadel_risk` best balanced threshold: thr=92.000, TPR=0.17, FPR=0.00, TP=4, FP=0, TN=8, FN=20
- `sig_wavlm_paralinguistic` best balanced threshold: thr=0.000, TPR=1.00, FPR=1.00, TP=24, FP=8, TN=0, FN=0
- `sig_rmt_divergence` best balanced threshold: thr=32.000, TPR=0.83, FPR=0.62, TP=20, FP=5, TN=3, FN=4

## Largest Coefficients

| feature | weight |
|---|---:|
| `bias` | 2.952 |
| `flatness_std` | -0.944 |
| `pitch_cv` | -0.911 |
| `speaker_cosine_vs_real_01` | -0.903 |
| `pitch_std` | -0.855 |
| `rms_std` | -0.766 |
| `silence_ratio` | -0.748 |
| `sig_asr_prefix_attack` | -0.583 |
| `pitch_mean` | -0.450 |
| `flatness_mean` | -0.376 |
| `sig_wavlm_paralinguistic` | -0.363 |
| `sig_rmt_divergence` | -0.347 |

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
    2.9519124769040137,
    -0.9027986587197445,
    -0.19395289103101188,
    -0.27007625477692204,
    -0.012215431182094663,
    -0.3632923244253295,
    -0.23705679037884403,
    0.08909534817042372,
    -0.3471817941894106,
    -0.5834199646754085,
    -0.25254758623030216,
    -0.7657856618202811,
    -0.7480420477479026,
    0.009470351601291565,
    -0.07704954988137674,
    -0.37589568627837056,
    -0.9442238114378736,
    0.0016310011756902026,
    0.04472741014312249,
    -0.45031481551205005,
    -0.8552141258965441,
    -0.911485990523165
  ],
  "mean": [
    0.7986641875000001,
    55.59375,
    2.3125,
    26.59375,
    59.15625,
    3.0,
    8.59375,
    52.09375,
    10.375,
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
    0.06643807776015834,
    26.205747669881507,
    0.5266343608235224,
    10.85374179430762,
    27.596545724737002,
    11.643131022195018,
    11.815930388145489,
    37.95915516627708,
    29.293503972724054,
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
  ]
}
```
