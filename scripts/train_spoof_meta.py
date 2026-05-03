#!/usr/bin/env python3
"""Train/evaluate a lightweight passive spoof meta-classifier."""

from __future__ import annotations

import argparse
import csv
import json
import math
from dataclasses import dataclass
from pathlib import Path
from typing import Any

import numpy as np


FEATURES = [
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
    "pitch_cv",
]

VOICE_FEATURES = ["speaker_cosine_vs_real_01"]
CITADEL_FEATURES = [
    "citadel_risk",
    "sig_spectral_anomaly",
    "sig_steganalysis",
    "sig_wavlm_paralinguistic",
    "sig_deepfake_detection",
    "sig_waveform_integrity",
    "sig_rmt_divergence",
    "sig_asr_prefix_attack",
]
LOCAL_FEATURES = [
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
FEATURE_SETS = {
    "all": FEATURES,
    "local+speaker_fast": VOICE_FEATURES + LOCAL_FEATURES,
    "local_only_fast": LOCAL_FEATURES,
    "citadel_only": CITADEL_FEATURES,
    "speaker_only": VOICE_FEATURES,
}


@dataclass(frozen=True)
class EvalResult:
    mode: str
    description: str
    scores: np.ndarray
    valid_mask: np.ndarray
    train_idx: np.ndarray
    test_idx: np.ndarray


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--features", default="docs/spoof_features.csv")
    parser.add_argument("--out", default="docs/SPOOF_META_RESULTS.md")
    parser.add_argument("--eval-mode", choices=["loo", "split"], default="loo", help="Primary holdout mode. Use split for public corpora.")
    parser.add_argument("--test-size", type=float, default=0.25, help="Per-label holdout fraction for --eval-mode split.")
    parser.add_argument("--seed", type=int, default=7, help="Deterministic split seed.")
    parser.add_argument("--group-by", action="append", default=[], help="Optional metadata column for leave-one-group-out evaluation. Can be repeated.")
    parser.add_argument("--group-holdout-mode", choices=["logo", "split"], default="logo", help="Group holdout strategy for --group-by columns.")
    args = parser.parse_args()

    rows = load_rows(Path(args.features))
    x_raw = np.asarray([[num(row[f]) for f in FEATURES] for row in rows], dtype=np.float64)
    y = np.asarray([1 if row["label"] == "synthetic" else 0 for row in rows], dtype=np.int64)

    eval_result = evaluate_scores(x_raw, y, args.eval_mode, args.test_size, args.seed)
    in_scores, weights, mean, std = train_and_score(x_raw, y, x_raw)
    threshold_report = best_thresholds(eval_result.scores[eval_result.valid_mask], y[eval_result.valid_mask])
    baseline_report = baseline_metrics(rows)
    ablations = feature_set_reports(rows, y, eval_result)
    group_reports = group_holdout_reports(rows, x_raw, y, args.group_by, args.group_holdout_mode, args.test_size, args.seed)

    out_path = Path(args.out)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(render_report(rows, y, eval_result, in_scores, weights, mean, std, threshold_report, baseline_report, ablations, group_reports))
    print(f"wrote {out_path}")
    print(threshold_report["summary"])
    return 0


def load_rows(path: Path) -> list[dict[str, str]]:
    with path.open() as f:
        return list(csv.DictReader(f))


def leave_one_out_scores(x: np.ndarray, y: np.ndarray) -> np.ndarray:
    scores = np.zeros(len(y), dtype=np.float64)
    for i in range(len(y)):
        train_idx = np.asarray([j for j in range(len(y)) if j != i], dtype=np.int64)
        test_idx = np.asarray([i], dtype=np.int64)
        scores[test_idx], *_ = train_and_score(x[train_idx], y[train_idx], x[test_idx])
    return scores


def evaluate_scores(x: np.ndarray, y: np.ndarray, mode: str, test_size: float, seed: int) -> EvalResult:
    if mode == "loo":
        scores = leave_one_out_scores(x, y)
        idx = np.arange(len(y), dtype=np.int64)
        return EvalResult(
            mode="loo",
            description="leave-one-out. This is useful for tiny calibration sets, but it is not scalable for public corpora.",
            scores=scores,
            valid_mask=np.ones(len(y), dtype=bool),
            train_idx=idx,
            test_idx=idx,
        )
    train_idx, test_idx = stratified_split_indices(y, test_size, seed)
    scores = split_holdout_scores(x, y, train_idx, test_idx)
    return EvalResult(
        mode="split",
        description=f"stratified holdout split, test_size={test_size:.2f}, seed={seed}.",
        scores=scores,
        valid_mask=np.isfinite(scores),
        train_idx=train_idx,
        test_idx=test_idx,
    )


def split_holdout_scores(x: np.ndarray, y: np.ndarray, train_idx: np.ndarray, test_idx: np.ndarray) -> np.ndarray:
    scores = np.full(len(y), np.nan, dtype=np.float64)
    scores[test_idx], *_ = train_and_score(x[train_idx], y[train_idx], x[test_idx])
    return scores


def stratified_split_indices(y: np.ndarray, test_size: float, seed: int) -> tuple[np.ndarray, np.ndarray]:
    if not 0.0 < test_size < 1.0:
        raise SystemExit("--test-size must be between 0 and 1")
    rng = np.random.default_rng(seed)
    train_parts = []
    test_parts = []
    for label in sorted(set(y.tolist())):
        idx = np.where(y == label)[0].astype(np.int64)
        if len(idx) < 2:
            raise SystemExit(f"not enough samples for stratified split label={label}: need at least 2")
        rng.shuffle(idx)
        n_test = max(1, int(math.ceil(len(idx) * test_size)))
        n_test = min(n_test, len(idx) - 1)
        test_parts.append(idx[:n_test])
        train_parts.append(idx[n_test:])
    train_idx = np.sort(np.concatenate(train_parts)).astype(np.int64)
    test_idx = np.sort(np.concatenate(test_parts)).astype(np.int64)
    if len(set(y[train_idx].tolist())) < 2 or len(set(y[test_idx].tolist())) < 2:
        raise SystemExit("stratified split did not preserve both classes in train and test")
    return train_idx, test_idx


def leave_one_group_out_scores(x: np.ndarray, y: np.ndarray, groups: list[str]) -> tuple[np.ndarray, list[str]]:
    scores = np.full(len(y), np.nan, dtype=np.float64)
    skipped: list[str] = []
    for group in sorted(set(groups)):
        test_idx = np.asarray([i for i, value in enumerate(groups) if value == group], dtype=np.int64)
        train_idx = np.asarray([i for i, value in enumerate(groups) if value != group], dtype=np.int64)
        if len(set(y[train_idx].tolist())) < 2:
            skipped.append(group)
            continue
        scores[test_idx], *_ = train_and_score(x[train_idx], y[train_idx], x[test_idx])
    return scores, skipped


def split_group_holdout_scores(
    x: np.ndarray,
    y: np.ndarray,
    groups: list[str],
    test_size: float,
    seed: int,
) -> tuple[np.ndarray, list[str], str]:
    unique_groups = sorted(set(groups))
    if len(unique_groups) < 2:
        return np.full(len(y), np.nan, dtype=np.float64), [], "need at least two groups"
    if not 0.0 < test_size < 1.0:
        return np.full(len(y), np.nan, dtype=np.float64), [], "--test-size must be between 0 and 1"

    n_test_groups = max(1, int(math.ceil(len(unique_groups) * test_size)))
    n_test_groups = min(n_test_groups, len(unique_groups) - 1)
    base_rng = np.random.default_rng(seed)
    for _ in range(200):
        shuffled = unique_groups[:]
        base_rng.shuffle(shuffled)
        test_groups = set(shuffled[:n_test_groups])
        test_idx = np.asarray([i for i, group in enumerate(groups) if group in test_groups], dtype=np.int64)
        train_idx = np.asarray([i for i, group in enumerate(groups) if group not in test_groups], dtype=np.int64)
        if len(test_idx) == 0 or len(train_idx) == 0:
            continue
        if len(set(y[train_idx].tolist())) < 2 or len(set(y[test_idx].tolist())) < 2:
            continue
        return split_holdout_scores(x, y, train_idx, test_idx), sorted(test_groups), ""
    return np.full(len(y), np.nan, dtype=np.float64), [], "could not find a group split with both classes in train and test"


def train_and_score(x_train_raw: np.ndarray, y_train: np.ndarray, x_score_raw: np.ndarray) -> tuple[np.ndarray, np.ndarray, np.ndarray, np.ndarray]:
    x_train, mean, std = standardize_fit(x_train_raw)
    x_score = standardize_apply(x_score_raw, mean, std)
    x_train = np.c_[np.ones(len(x_train)), x_train]
    x_score = np.c_[np.ones(len(x_score)), x_score]
    weights = fit_logistic(x_train, y_train)
    scores = sigmoid(x_score @ weights)
    return scores, weights, mean, std


def score_feature_set(rows: list[dict[str, str]], features: list[str], y: np.ndarray, eval_result: EvalResult) -> np.ndarray:
    x_raw = np.asarray([[num(row[f]) for f in features] for row in rows], dtype=np.float64)
    if eval_result.mode == "split":
        return split_holdout_scores(x_raw, y, eval_result.train_idx, eval_result.test_idx)
    return leave_one_out_scores(x_raw, y)


def fit_logistic(x: np.ndarray, y: np.ndarray, steps: int = 2500, lr: float = 0.08, l2: float = 0.25) -> np.ndarray:
    weights = np.zeros(x.shape[1], dtype=np.float64)
    for _ in range(steps):
        pred = sigmoid(x @ weights)
        grad = x.T @ (pred - y) / len(y)
        reg = l2 * weights / len(y)
        reg[0] = 0.0
        weights -= lr * (grad + reg)
    return weights


def standardize_fit(x: np.ndarray) -> tuple[np.ndarray, np.ndarray, np.ndarray]:
    mean = x.mean(axis=0)
    std = x.std(axis=0)
    std[std < 1e-9] = 1.0
    return (x - mean) / std, mean, std


def standardize_apply(x: np.ndarray, mean: np.ndarray, std: np.ndarray) -> np.ndarray:
    return (x - mean) / std


def sigmoid(z: np.ndarray) -> np.ndarray:
    z = np.clip(z, -40, 40)
    return 1.0 / (1.0 + np.exp(-z))


def best_thresholds(scores: np.ndarray, y: np.ndarray) -> dict[str, Any]:
    thresholds = sorted(set(float(s) for s in scores))
    candidates = []
    for threshold in thresholds:
        pred = scores >= threshold
        tp = int(((pred == 1) & (y == 1)).sum())
        fp = int(((pred == 1) & (y == 0)).sum())
        tn = int(((pred == 0) & (y == 0)).sum())
        fn = int(((pred == 0) & (y == 1)).sum())
        tpr = tp / max(1, tp + fn)
        fpr = fp / max(1, fp + tn)
        candidates.append({"threshold": threshold, "tp": tp, "fp": fp, "tn": tn, "fn": fn, "tpr": tpr, "fpr": fpr})
    zero_fp = [c for c in candidates if c["fp"] == 0]
    best_zero_fp = max(zero_fp, key=lambda c: c["tpr"]) if zero_fp else None
    best_balanced = max(candidates, key=lambda c: (c["tpr"] - c["fpr"], c["tpr"]))
    summary = (
        f"best_zero_fp={fmt_candidate(best_zero_fp)}; "
        f"best_balanced={fmt_candidate(best_balanced)}"
    )
    return {"candidates": candidates, "best_zero_fp": best_zero_fp, "best_balanced": best_balanced, "summary": summary}


def baseline_metrics(rows: list[dict[str, str]]) -> dict[str, Any]:
    y = np.asarray([1 if row["label"] == "synthetic" else 0 for row in rows])
    out = {}
    for field in ["sig_deepfake_detection", "citadel_risk", "sig_wavlm_paralinguistic", "sig_rmt_divergence"]:
        scores = np.asarray([num(row[field]) for row in rows], dtype=np.float64)
        out[field] = best_thresholds(scores, y)["best_balanced"]
    return out


def feature_set_reports(rows: list[dict[str, str]], y: np.ndarray, eval_result: EvalResult) -> dict[str, dict[str, Any]]:
    out = {}
    for name, features in FEATURE_SETS.items():
        scores = score_feature_set(rows, features, y, eval_result)
        valid = np.isfinite(scores)
        out[name] = {
            "features": features,
            "scores": scores,
            "thresholds": best_thresholds(scores[valid], y[valid]),
            "calibration": calibration(scores[valid], y[valid]),
        }
    return out


def group_holdout_reports(
    rows: list[dict[str, str]],
    x: np.ndarray,
    y: np.ndarray,
    columns: list[str],
    mode: str,
    test_size: float,
    seed: int,
) -> dict[str, dict[str, Any]]:
    reports = {}
    if not rows:
        return reports
    for column in columns:
        if column not in rows[0]:
            reports[column] = {"error": "column missing", "mode": mode}
            continue
        groups = [row.get(column, "") or "unspecified" for row in rows]
        if mode == "split":
            scores, held_out_groups, error = split_group_holdout_scores(x, y, groups, test_size, seed)
            skipped = []
            if error:
                reports[column] = {"error": error, "skipped": skipped, "held_out_groups": held_out_groups, "mode": mode}
                continue
        else:
            scores, skipped = leave_one_group_out_scores(x, y, groups)
            held_out_groups = []
        valid = ~np.isnan(scores)
        if int(valid.sum()) == 0:
            reports[column] = {"error": "no valid folds", "skipped": skipped, "mode": mode}
            continue
        reports[column] = {
            "scores": scores,
            "skipped": skipped,
            "coverage": int(valid.sum()),
            "groups": sorted(set(groups)),
            "held_out_groups": held_out_groups,
            "mode": mode,
            "thresholds": best_thresholds(scores[valid], y[valid]),
            "calibration": calibration(scores[valid], y[valid]),
        }
    return reports


def calibration(scores: np.ndarray, y: np.ndarray) -> dict[str, Any]:
    real_scores = scores[y == 0]
    synth_scores = scores[y == 1]
    max_real = float(real_scores.max()) if len(real_scores) else 1.0
    min_synth = float(synth_scores.min()) if len(synth_scores) else 0.0
    above_real = [float(s) for s in synth_scores if s > max_real]
    nearest_detected = min(above_real) if above_real else None
    block_threshold = (max_real + nearest_detected) / 2 if nearest_detected is not None else max_real + 1e-6
    all_synth_threshold = min_synth
    warn_threshold = min(all_synth_threshold, max(0.0, block_threshold - 0.05))
    return {
        "max_real": max_real,
        "min_synthetic": min_synth,
        "nearest_synthetic_above_real": nearest_detected,
        "suggested_warn_threshold": warn_threshold,
        "suggested_block_threshold": block_threshold,
        "block_metrics": metrics_at(scores, y, block_threshold),
        "catch_all_metrics": metrics_at(scores, y, all_synth_threshold),
    }


def metrics_at(scores: np.ndarray, y: np.ndarray, threshold: float) -> dict[str, Any]:
    pred = scores >= threshold
    tp = int(((pred == 1) & (y == 1)).sum())
    fp = int(((pred == 1) & (y == 0)).sum())
    tn = int(((pred == 0) & (y == 0)).sum())
    fn = int(((pred == 0) & (y == 1)).sum())
    return {
        "threshold": threshold,
        "tp": tp,
        "fp": fp,
        "tn": tn,
        "fn": fn,
        "tpr": tp / max(1, tp + fn),
        "fpr": fp / max(1, fp + tn),
    }


def render_report(
    rows: list[dict[str, str]],
    y: np.ndarray,
    eval_result: EvalResult,
    in_scores: np.ndarray,
    weights: np.ndarray,
    mean: np.ndarray,
    std: np.ndarray,
    threshold_report: dict[str, Any],
    baseline_report: dict[str, Any],
    ablations: dict[str, dict[str, Any]],
    group_reports: dict[str, dict[str, Any]],
) -> str:
    lines = [
        "# Spoof Meta-Detector Experiment",
        "",
        f"Samples: {len(rows)} ({int((y == 0).sum())} real, {int((y == 1).sum())} synthetic)",
        "",
        "## Holdout Result",
        "",
        f"- Evaluation: {eval_result.description}",
        f"- Evaluated samples: {int(eval_result.valid_mask.sum())}/{len(rows)}.",
        f"- {threshold_report['summary']}",
        "",
        "## Scores",
        "",
        "| file | label | split | holdout spoof score | in-sample score | current deepfake | current overall | latency ms |",
        "|---|---:|---:|---:|---:|---:|---:|---:|",
    ]
    score_rows = sorted(zip(rows, eval_result.scores, in_scores, strict=True), key=score_sort_key)
    for row, holdout, ins in score_rows:
        split = "eval" if math.isfinite(float(holdout)) else "train"
        if eval_result.mode == "loo":
            split = "loo"
        lines.append(
            f"| `{Path(row['file']).name}` | {row['label']} | {split} | {fmt_score(holdout)} | {ins:.3f} | {row['sig_deepfake_detection']} | {row['citadel_risk']} | {row['citadel_latency_ms']} |"
        )
    holdout_scores = eval_result.scores[eval_result.valid_mask]
    holdout_y = y[eval_result.valid_mask]
    main_cal = calibration(holdout_scores, holdout_y)
    lines.extend(
        [
            "",
            "## Operating Bands",
            "",
            "- Scores are calibrated on held-out predictions, not in-sample predictions.",
            f"- Max real score: {main_cal['max_real']:.3f}; min synthetic score: {main_cal['min_synthetic']:.3f}; nearest detected synthetic above the real range: {fmt_optional(main_cal['nearest_synthetic_above_real'])}.",
            f"- Suggested legacy WARN threshold: {main_cal['suggested_warn_threshold']:.3f}. Suggested legacy BLOCK threshold: {main_cal['suggested_block_threshold']:.3f}.",
            f"- At BLOCK threshold: {fmt_candidate(main_cal['block_metrics'])}.",
            f"- At catch-all synthetic threshold: {fmt_candidate(main_cal['catch_all_metrics'])}.",
            "",
            "## Feature-Set Ablations",
            "",
            "| feature set | zero-FP result | max real | min synthetic | suggested block | catch-all result |",
            "|---|---|---:|---:|---:|---|",
        ]
    )
    for name, report in ablations.items():
        cal = report["calibration"]
        zero_fp = report["thresholds"]["best_zero_fp"]
        lines.append(
            f"| `{name}` | {fmt_candidate(zero_fp)} | {cal['max_real']:.3f} | {cal['min_synthetic']:.3f} | {cal['suggested_block_threshold']:.3f} | {fmt_candidate(cal['catch_all_metrics'])} |"
        )
    if group_reports:
        lines.extend(
            [
                "",
                "## Group Holdouts",
                "",
                "| held-out column | mode | groups | covered samples | skipped groups | zero-FP result |",
                "|---|---|---:|---:|---:|---|",
            ]
        )
        for column, report in group_reports.items():
            if "error" in report:
                lines.append(f"| `{column}` | {report.get('mode', '')} | 0 | 0 | {len(report.get('skipped', []))} | {report['error']} |")
                continue
            lines.append(
                f"| `{column}` | {report.get('mode', '')} | {len(report['groups'])} | {report['coverage']} | {len(report['skipped'])} | {fmt_candidate(report['thresholds']['best_zero_fp'])} |"
            )
    lines.extend(["", "## Current-Signal Baselines", ""])
    for field, candidate in baseline_report.items():
        lines.append(f"- `{field}` best balanced threshold: {fmt_candidate(candidate)}")
    lines.extend(["", "## Largest Coefficients", "", "| feature | weight |", "|---|---:|"])
    pairs = list(zip(["bias", *FEATURES], weights, strict=True))
    for name, weight in sorted(pairs, key=lambda p: abs(p[1]), reverse=True)[:12]:
        lines.append(f"| `{name}` | {weight:.3f} |")
    lines.extend(
        [
            "",
            "## Interpretation",
            "",
            "This experiment can show whether the available features contain signal, but it does not prove generalization. A deployable detector needs more real speakers, more clone providers, and provider-held-out evaluation.",
            "",
        ]
    )
    artifact = {"features": FEATURES, "weights": weights.tolist(), "mean": mean.tolist(), "std": std.tolist(), "evaluation": eval_result.description}
    lines.extend(["## Model Artifact", "", "```json", json.dumps(artifact, indent=2), "```", ""])
    return "\n".join(lines)


def score_sort_key(item: tuple[dict[str, str], float, float]) -> tuple[int, float]:
    score = float(item[1])
    if math.isfinite(score):
        return (0, -score)
    return (1, 0.0)


def fmt_score(value: float) -> str:
    value = float(value)
    return "" if not math.isfinite(value) else f"{value:.3f}"


def fmt_candidate(candidate: dict[str, Any] | None) -> str:
    if candidate is None:
        return "none"
    return (
        f"thr={candidate['threshold']:.3f}, "
        f"TPR={candidate['tpr']:.2f}, FPR={candidate['fpr']:.2f}, "
        f"TP={candidate['tp']}, FP={candidate['fp']}, TN={candidate['tn']}, FN={candidate['fn']}"
    )


def fmt_optional(value: float | None) -> str:
    return "none" if value is None else f"{value:.3f}"


def num(value: Any) -> float:
    try:
        return float(value)
    except (TypeError, ValueError):
        return 0.0


if __name__ == "__main__":
    raise SystemExit(main())
