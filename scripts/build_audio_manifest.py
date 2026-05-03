#!/usr/bin/env python3
"""Build a normalized audio anti-spoofing manifest from OSS dataset layouts."""

from __future__ import annotations

import argparse
import csv
import os
from collections import Counter, defaultdict
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Iterable


AUDIO_SUFFIXES = {".wav", ".flac", ".mp3", ".m4a", ".ogg", ".opus", ".aif", ".aiff"}
MANIFEST_FIELDS = [
    "path",
    "label",
    "dataset",
    "speaker_id",
    "language",
    "generator_id",
    "attack_type",
    "codec",
    "sample_rate",
    "license",
    "split",
    "duration_s",
    "text",
    "source_url",
]
DEFAULT_LICENSES = {
    "asvspoof2019": "odc-by",
    "asvspoof2021": "odc-by",
    "fixtures-user": "private-local",
    "generic": "unknown",
    "in-the-wild": "cc-by-sa-4.0",
    "mlaad": "cc-by-nc-4.0",
    "multiapi": "cc-by-nc-4.0",
    "sine": "apache-2.0",
    "wavefake": "research",
}
DEFAULT_SOURCE_URLS = {
    "asvspoof2019": "https://www.asvspoof.org/",
    "asvspoof2021": "https://www.asvspoof.org/",
    "in-the-wild": "https://huggingface.co/datasets/mueller91/In-The-Wild",
    "mlaad": "https://huggingface.co/datasets/mueller91/MLAAD",
    "multiapi": "https://huggingface.co/datasets/jhsdfbsdjfu/MultiAPI-Spoof",
    "sine": "https://huggingface.co/datasets/PeacefulData/SINE",
    "wavefake": "https://zenodo.org/records/5642694",
}


@dataclass
class AudioRecord:
    path: str
    label: str
    dataset: str
    speaker_id: str = ""
    language: str = ""
    generator_id: str = ""
    attack_type: str = ""
    codec: str = ""
    sample_rate: str = ""
    license: str = ""
    split: str = ""
    duration_s: str = ""
    text: str = ""
    source_url: str = ""


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--adapter", choices=sorted(DEFAULT_LICENSES), required=True)
    parser.add_argument("--root", required=True, help="Downloaded/cloned dataset root.")
    parser.add_argument("--out", default="data/audio_manifest.csv")
    parser.add_argument("--append", action="store_true", help="Append to an existing manifest.")
    parser.add_argument("--allow-missing", action="store_true", help="Keep metadata rows whose audio file is not present locally.")
    parser.add_argument("--max-per-label", type=int, default=0, help="Optional cap per label for smoke tests.")
    parser.add_argument("--dataset-name", default="")
    parser.add_argument("--label", default="", help="Optional label override for single-class generic folders.")
    parser.add_argument("--speaker-id", default="")
    parser.add_argument("--language", default="")
    parser.add_argument("--license", default="")
    parser.add_argument("--split", default="")
    parser.add_argument("--source-url", default="")
    parser.add_argument("--metadata-file", default="", help="For adapter=generic: CSV/TSV metadata path.")
    parser.add_argument("--path-column", default="path", help="For adapter=generic with metadata.")
    parser.add_argument("--label-column", default="label", help="For adapter=generic with metadata.")
    parser.add_argument("--delimiter", default="", help="For adapter=generic with metadata. Defaults to sniffing.")
    args = parser.parse_args()

    root = Path(args.root).expanduser()
    if not root.exists():
        raise SystemExit(f"root does not exist: {root}")

    records = build_records(args.adapter, root, args)
    if not args.allow_missing:
        records = [r for r in records if audio_exists(r.path, root)]
    if args.max_per_label > 0:
        records = cap_per_label(records, args.max_per_label)

    if args.append and Path(args.out).exists():
        records = read_manifest(Path(args.out)) + records
    records = dedupe(records)
    write_manifest(Path(args.out), records)
    print_summary(records, Path(args.out))
    return 0


def build_records(adapter: str, root: Path, args: argparse.Namespace) -> list[AudioRecord]:
    if adapter == "fixtures-user":
        return fixtures_user_records(root, args)
    if adapter in {"asvspoof2019", "asvspoof2021"}:
        return asvspoof_records(root, args)
    if adapter == "mlaad":
        return mlaad_records(root, args)
    if adapter == "multiapi":
        return multiapi_records(root, args)
    if adapter == "sine":
        return sine_records(root, args)
    if adapter == "in-the-wild":
        return in_the_wild_records(root, args)
    if adapter == "wavefake":
        return inferred_tree_records(root, args, dataset_default="wavefake")
    if adapter == "generic":
        if args.metadata_file:
            return generic_metadata_records(root, args)
        return inferred_tree_records(root, args, dataset_default=args.dataset_name or root.name)
    raise AssertionError(f"unhandled adapter: {adapter}")


def fixtures_user_records(root: Path, args: argparse.Namespace) -> list[AudioRecord]:
    records = []
    for path in iter_audio(root):
        source = fixture_source(path)
        label = "real" if source == "real" else "synthetic"
        records.append(
            base_record(
                path,
                args,
                dataset_default=args.dataset_name or "hampshire_user",
                label=label,
                speaker_id=args.speaker_id or "target_user",
                generator_id="" if label == "real" else source,
                attack_type="" if label == "real" else "voice_clone",
                split=args.split or "eval",
            )
        )
    return records


def asvspoof_records(root: Path, args: argparse.Namespace) -> list[AudioRecord]:
    audio_index = {p.stem: p for p in iter_audio(root)}
    records = []
    for protocol in root.rglob("*.txt"):
        if "README" in protocol.name.upper():
            continue
        split = args.split or infer_split(protocol)
        attack_type = infer_asvspoof_attack(protocol)
        for raw in protocol.read_text(errors="ignore").splitlines():
            fields = raw.strip().split()
            if len(fields) < 2:
                continue
            label_token = next((f for f in reversed(fields) if normalize_label(f)), "")
            label = normalize_label(label_token)
            if not label:
                continue
            audio_id = next((f for f in fields if f in audio_index), "")
            if not audio_id and len(fields) > 1:
                audio_id = fields[1]
            path = audio_index.get(audio_id)
            if path is None:
                continue
            records.append(
                base_record(
                    path,
                    args,
                    dataset_default=args.dataset_name or args.adapter,
                    label=label,
                    speaker_id=fields[0],
                    generator_id="" if label == "real" else value_or_blank(fields, 2),
                    attack_type=attack_type,
                    split=split,
                )
            )
    return records


def mlaad_records(root: Path, args: argparse.Namespace) -> list[AudioRecord]:
    records = []
    for meta in root.rglob("meta.csv"):
        for row in read_table(meta, delimiter="|"):
            rel = row.get("path") or row.get("audio") or row.get("file") or row.get("filename")
            if not rel:
                continue
            path = resolve_metadata_path(meta.parent, rel)
            records.append(
                base_record(
                    path,
                    args,
                    dataset_default=args.dataset_name or "mlaad",
                    label="synthetic",
                    speaker_id=row.get("reference_speaker", ""),
                    language=row.get("language", args.language),
                    generator_id=row.get("model_name", ""),
                    attack_type=row.get("architecture", "tts"),
                    split=args.split or infer_split(meta),
                    text=row.get("transcript", ""),
                )
            )
    return records


def multiapi_records(root: Path, args: argparse.Namespace) -> list[AudioRecord]:
    records = []
    for meta in root.rglob("*.txt"):
        split = args.split or infer_split(meta)
        for raw in meta.read_text(errors="ignore").splitlines():
            line = raw.strip()
            if not line or line.startswith("#"):
                continue
            fields = line.split()
            if len(fields) < 3:
                continue
            label = normalize_label(fields[-1])
            if not label:
                continue
            path = resolve_metadata_path(root, fields[0])
            api = fields[-2] if fields[-2] != "-" else ""
            records.append(
                base_record(
                    path,
                    args,
                    dataset_default=args.dataset_name or "multiapi_spoof",
                    label=label,
                    generator_id=api,
                    attack_type="multi_source_tts" if label == "synthetic" else "",
                    split=split,
                )
            )
    return records


def in_the_wild_records(root: Path, args: argparse.Namespace) -> list[AudioRecord]:
    meta = next(root.rglob("meta.csv"), None)
    if meta is None:
        return inferred_tree_records(root, args, dataset_default=args.dataset_name or "in_the_wild")
    records = []
    for row in read_table(meta):
        rel = row.get("file") or row.get("path") or row.get("filename")
        if not rel:
            continue
        path = resolve_metadata_path(meta.parent, rel)
        label = normalize_label(row.get("label", "")) or "unknown"
        records.append(
            base_record(
                path,
                args,
                dataset_default=args.dataset_name or "in_the_wild",
                label=label,
                speaker_id=row.get("speaker", ""),
                language=args.language or "en",
                generator_id="unknown_public_web" if label == "synthetic" else "",
                attack_type="in_the_wild_tts" if label == "synthetic" else "authentic_public_web",
                split=args.split or "eval",
            )
        )
    return records


def sine_records(root: Path, args: argparse.Namespace) -> list[AudioRecord]:
    records = []
    for path in iter_audio(root):
        lower = str(path).lower()
        if "cut_paste" in lower or "dev_edit" in lower or "/edit" in lower:
            label = "synthetic"
            attack_type = "speech_edit"
        elif "resyn" in lower:
            label = "real"
            attack_type = "resynthesized_authentic"
        elif "real" in lower:
            label = "real"
            attack_type = "authentic"
        else:
            label = normalize_label(args.label) or "unknown"
            attack_type = ""
        records.append(
            base_record(
                path,
                args,
                dataset_default=args.dataset_name or "sine",
                label=label,
                attack_type=attack_type,
                split=args.split or infer_split(path),
            )
        )
    return records


def inferred_tree_records(root: Path, args: argparse.Namespace, dataset_default: str) -> list[AudioRecord]:
    records = []
    for path in iter_audio(root):
        label = normalize_label(args.label) or infer_label_from_path(path)
        if label == "unknown" and args.dataset_name:
            label = infer_label_from_path(path)
        records.append(
            base_record(
                path,
                args,
                dataset_default=args.dataset_name or dataset_default,
                label=label,
                generator_id=infer_generator_from_path(path),
                attack_type=infer_attack_from_path(path),
                split=args.split or infer_split(path),
            )
        )
    return records


def generic_metadata_records(root: Path, args: argparse.Namespace) -> list[AudioRecord]:
    metadata = Path(args.metadata_file).expanduser()
    if not metadata.is_absolute():
        metadata = root / metadata
    records = []
    for row in read_table(metadata, delimiter=args.delimiter or None):
        rel = row.get(args.path_column, "")
        if not rel:
            continue
        label = normalize_label(row.get(args.label_column, "")) or "unknown"
        path = resolve_metadata_path(root, rel)
        records.append(
            base_record(
                path,
                args,
                dataset_default=args.dataset_name or root.name,
                label=label,
                speaker_id=row.get("speaker_id", row.get("speaker", args.speaker_id)),
                language=row.get("language", args.language),
                generator_id=row.get("generator_id", row.get("model", row.get("api", ""))),
                attack_type=row.get("attack_type", row.get("attack", "")),
                split=row.get("split", args.split or infer_split(path)),
                text=row.get("text", row.get("transcript", "")),
            )
        )
    return records


def base_record(
    path: Path,
    args: argparse.Namespace,
    dataset_default: str,
    label: str,
    speaker_id: str = "",
    language: str = "",
    generator_id: str = "",
    attack_type: str = "",
    split: str = "",
    text: str = "",
) -> AudioRecord:
    return AudioRecord(
        path=portable_path(path),
        label=label,
        dataset=args.dataset_name or dataset_default,
        speaker_id=speaker_id or args.speaker_id,
        language=language or args.language,
        generator_id=generator_id,
        attack_type=attack_type,
        codec=path.suffix.lower().lstrip("."),
        license=args.license or DEFAULT_LICENSES.get(args.adapter, "unknown"),
        split=split,
        text=text,
        source_url=args.source_url or DEFAULT_SOURCE_URLS.get(args.adapter, ""),
    )


def read_table(path: Path, delimiter: str | None = None) -> list[dict[str, str]]:
    text = path.read_text(errors="ignore")
    if delimiter is None:
        sample = text[:2048]
        delimiter = csv.Sniffer().sniff(sample, delimiters=",\t|;").delimiter if sample.strip() else ","
    with path.open(newline="", errors="ignore") as f:
        return list(csv.DictReader(f, delimiter=delimiter))


def read_manifest(path: Path) -> list[AudioRecord]:
    with path.open(newline="") as f:
        return [AudioRecord(**{field: row.get(field, "") for field in MANIFEST_FIELDS}) for row in csv.DictReader(f)]


def write_manifest(path: Path, records: list[AudioRecord]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=MANIFEST_FIELDS)
        writer.writeheader()
        for record in records:
            writer.writerow(asdict(record))


def print_summary(records: list[AudioRecord], out_path: Path) -> None:
    print(f"wrote {out_path} ({len(records)} rows)")
    for name, counter in [
        ("label", Counter(r.label for r in records)),
        ("dataset", Counter(r.dataset for r in records)),
        ("split", Counter(r.split or "unspecified" for r in records)),
    ]:
        print(f"{name}: " + ", ".join(f"{k}={v}" for k, v in sorted(counter.items())))


def cap_per_label(records: Iterable[AudioRecord], max_per_label: int) -> list[AudioRecord]:
    counts: defaultdict[str, int] = defaultdict(int)
    capped = []
    for record in records:
        if counts[record.label] >= max_per_label:
            continue
        capped.append(record)
        counts[record.label] += 1
    return capped


def dedupe(records: Iterable[AudioRecord]) -> list[AudioRecord]:
    seen = set()
    out = []
    for record in records:
        key = (record.dataset, record.path, record.label)
        if key in seen:
            continue
        seen.add(key)
        out.append(record)
    return out


def iter_audio(root: Path) -> Iterable[Path]:
    return sorted(p for p in root.rglob("*") if p.is_file() and p.suffix.lower() in AUDIO_SUFFIXES)


def audio_exists(path_value: str, root: Path) -> bool:
    path = Path(path_value)
    return path.is_file() or (root / path).is_file() or (Path.cwd() / path).is_file()


def portable_path(path: Path) -> str:
    expanded = path.expanduser()
    resolved = expanded.resolve()
    try:
        return str(resolved.relative_to(Path.cwd().resolve()))
    except ValueError:
        return str(expanded if expanded.is_absolute() else resolved)


def resolve_metadata_path(base: Path, value: str) -> Path:
    path = Path(value)
    return path if path.is_absolute() else base / path


def fixture_source(path: Path) -> str:
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


def normalize_label(value: str | None) -> str:
    token = str(value or "").strip().lower()
    if token in {"bonafide", "bona-fide", "genuine", "human", "real", "authentic", "true", "1"}:
        return "real"
    if token in {"spoof", "spoofed", "fake", "synthetic", "generated", "deepfake", "manipulated", "false", "0"}:
        return "synthetic"
    return ""


def infer_label_from_path(path: Path) -> str:
    lower = "/" + str(path).lower().replace(os.sep, "/") + "/"
    if any(token in lower for token in ["/bonafide/", "/genuine/", "/real/", "/authentic/", "/dev_real"]):
        return "real"
    if any(token in lower for token in ["/spoof/", "/spoofed/", "/fake/", "/synthetic/", "/generated/", "/clone/", "/dev_edit", "/cut_paste"]):
        return "synthetic"
    return "unknown"


def infer_generator_from_path(path: Path) -> str:
    lower = str(path).lower()
    for token in ["cartesia", "elevenlabs", "hifigan", "hifi-gan", "melgan", "waveglow", "wavernn", "parallel_wavegan", "bark", "xtts", "rvc"]:
        if token in lower:
            return token
    return ""


def infer_attack_from_path(path: Path) -> str:
    lower = str(path).lower()
    if "clone" in lower:
        return "voice_clone"
    if "tts" in lower or "generated" in lower or "fake" in lower or "spoof" in lower:
        return "tts"
    if "edit" in lower or "cut_paste" in lower:
        return "speech_edit"
    if "replay" in lower:
        return "replay"
    return ""


def infer_split(path: Path) -> str:
    lower = str(path).lower()
    if "train" in lower:
        return "train"
    if "dev" in lower or "valid" in lower:
        return "dev"
    if "eval" in lower or "test" in lower:
        return "eval"
    return ""


def infer_asvspoof_attack(path: Path) -> str:
    lower = str(path).lower()
    if "/pa/" in lower or "_pa_" in lower or "physical" in lower:
        return "physical_access"
    if "/df/" in lower or "_df_" in lower or "deepfake" in lower:
        return "deepfake"
    if "/la/" in lower or "_la_" in lower or "logical" in lower:
        return "logical_access"
    return "spoof"


def value_or_blank(values: list[str], index: int) -> str:
    if index >= len(values) or values[index] == "-":
        return ""
    return values[index]


if __name__ == "__main__":
    raise SystemExit(main())
