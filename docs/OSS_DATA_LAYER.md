# OSS Audio Data Layer

This layer normalizes public and private audio anti-spoofing corpora into one CSV manifest so the passive spoof detector can be trained and evaluated with strict dataset/provider holdouts.

Large audio should live in Modal volumes, not this repo. Keep only manifests, small reports, and scripts locally.

## Manifest Schema

Every dataset adapter writes:

```text
path,label,dataset,speaker_id,language,generator_id,attack_type,codec,sample_rate,license,split,duration_s,text,source_url
```

Required fields:

- `path`: local or absolute audio path. If the dataset is on Modal, this should point to the mounted volume path inside the Modal job.
- `label`: `real` or `synthetic`.
- `dataset`: source corpus, e.g. `asvspoof2019`, `mlaad`, `multiapi_spoof`.
- `generator_id`: TTS API/model/provider when known.
- `split`: `train`, `dev`, or `eval`.
- `license`: keep this populated. Some sources are non-commercial or gated.

## Supported Adapters

| Adapter | Source | Use |
|---|---|---|
| `fixtures-user` | local `fixtures/user` | Private target-speaker calibration and Cartesia hard negatives |
| `asvspoof2019` / `asvspoof2021` | ASVspoof releases or mirrors | Standard anti-spoof benchmark data |
| `mlaad` | Hugging Face `mueller91/MLAAD` | Multilingual TTS spoof data |
| `multiapi` | Hugging Face `jhsdfbsdjfu/MultiAPI-Spoof` | Commercial/open-source/web TTS provider diversity |
| `in-the-wild` | Hugging Face `mueller91/In-The-Wild` | Realistic celebrity/politician generalization eval |
| `wavefake` | Zenodo WaveFake | Classic vocoder/TTS artifacts |
| `sine` | Hugging Face `PeacefulData/SINE` | Speech edit/splice localization and manipulation |
| `generic` | any folder or metadata CSV | Escape hatch for new corpora |

## Local Smoke Test

```bash
cd natsec
make manifest
python3 scripts/extract_spoof_features.py \
  --manifest data/audio_manifest.csv \
  --skip-voicebio \
  --skip-citadel \
  --out docs/spoof_features_manifest_smoke.csv
python3 scripts/train_spoof_meta.py \
  --features docs/spoof_features_manifest_smoke.csv \
  --out docs/SPOOF_META_MANIFEST_SMOKE.md \
  --group-by dataset \
  --group-by generator_id
```

This smoke path uses local acoustic features only. It should remain fast enough to run in this constrained workspace.

For public corpora, do not use leave-one-out. Use a fixed stratified holdout and grouped holdout splits:

```bash
python3 scripts/train_spoof_meta.py \
  --features /data/mighty-morphing/audio/features/in_the_wild_local_features.csv \
  --out /data/mighty-morphing/audio/reports/in_the_wild_local_holdouts.md \
  --eval-mode split \
  --test-size 0.25 \
  --seed 7 \
  --group-holdout-mode split \
  --group-by dataset \
  --group-by generator_id \
  --group-by speaker_id
```

## Modal Storage

Default targets:

```text
MODAL_PROFILE=your-modal-profile
MODAL_VOLUME=citadel-audio-oss-data
MODAL_PREFIX=mighty-morphing/audio
```

Upload the current local fixture/report set:

```bash
cd natsec
make modal-volume
make modal-put-fixtures
make modal-ls
```

Current uploaded layout:

```text
mighty-morphing/audio/fixtures/user/
mighty-morphing/audio/manifests/audio_manifest.csv
mighty-morphing/audio/reports/detector_validation_results.csv
mighty-morphing/audio/reports/spoof_features.csv
mighty-morphing/audio/reports/SPOOF_META_RESULTS.md
mighty-morphing/audio/reports/spoof_features_manifest_smoke.csv
mighty-morphing/audio/reports/SPOOF_META_MANIFEST_SMOKE.md
mighty-morphing/audio/reports/SPOOF_META_MANIFEST_SPLIT_SMOKE.md
```

For large public datasets, download or clone directly in a Modal job or upload the archive/directory to this same volume. Do not stage 30-100GB corpora in the workspace.

The current Modal public smoke runner stages In-The-Wild directly on the volume, caps to 500 real and 500 synthetic samples, extracts local acoustic features, and writes a split-holdout report:

```bash
cd natsec
make modal-run-in-the-wild
```

## Example Public-Corpus Manifests

```bash
# MLAAD, after accepting the HF terms and cloning/downloading on a machine with space.
python3 scripts/build_audio_manifest.py \
  --adapter mlaad \
  --root /data/MLAAD \
  --out /data/manifests/mlaad.csv

# MultiAPI-Spoof.
python3 scripts/build_audio_manifest.py \
  --adapter multiapi \
  --root /data/MultiAPI-Spoof \
  --out /data/manifests/multiapi.csv

# ASVspoof 2019.
python3 scripts/build_audio_manifest.py \
  --adapter asvspoof2019 \
  --root /data/ASVspoof2019 \
  --out /data/manifests/asvspoof2019.csv

# Generic folder with inferred labels from path names like real/, fake/, spoof/.
python3 scripts/build_audio_manifest.py \
  --adapter generic \
  --root /data/new-corpus \
  --dataset-name new_corpus \
  --out /data/manifests/new_corpus.csv
```

## Evaluation Rule

Use public corpora for breadth, but do not trust pooled random splits. Use held-out groups:

```bash
python3 scripts/train_spoof_meta.py \
  --features /data/features/all_features.csv \
  --out /data/reports/spoof_meta_public_holdouts.md \
  --eval-mode split \
  --group-holdout-mode split \
  --group-by dataset \
  --group-by generator_id \
  --group-by language
```

The model is only credible if it holds up on unseen datasets, unseen generators, and weird real speech.
