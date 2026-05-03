MODAL_PROFILE ?= your-modal-profile
MODAL_VOLUME ?= citadel-audio-oss-data
MODAL_PREFIX ?= mighty-morphing/audio

.PHONY: help dev backend web voicebio setup tidy test manifest modal-volume modal-put-fixtures modal-run-in-the-wild modal-ls reset-demo clean

help:
	@echo "Mighty Morphing — common commands"
	@echo ""
	@echo "  make setup     install dependencies (bun + go + python venv)"
	@echo "  make dev       run voicebio + backend + web concurrently"
	@echo "  make voicebio  run Python voice-biometric sidecar only"
	@echo "  make backend   run Go gateway only"
	@echo "  make web       run Vite dev server only"
	@echo "  make tidy      go mod tidy + bun update lockfile"
	@echo "  make test      backend tests + web typecheck/build + script syntax checks"
	@echo "  make manifest  build local audio manifest from fixtures/user"
	@echo "  make modal-put-fixtures upload local audio fixtures + manifests to Modal volume"
	@echo "  make modal-run-in-the-wild run the 1k-row public-corpus Modal smoke pipeline"
	@echo "  make reset-demo clear local registry and audit state"
	@echo "  make clean     remove build artifacts"

setup:
	cd web && bun install
	cd backend && go mod tidy
	cd voicebio && python3.12 -m venv .venv && \
		.venv/bin/pip install --upgrade pip && \
		.venv/bin/pip install -r requirements.txt

dev:
	@echo "Starting voicebio :7100, backend :7000, web :5173..."
	@trap 'kill 0' INT TERM; \
		(cd voicebio && .venv/bin/uvicorn main:app --host 0.0.0.0 --port 7100) & \
		(cd backend && go run .) & \
		(cd web && bun run dev) & \
		wait

voicebio:
	cd voicebio && .venv/bin/uvicorn main:app --host 0.0.0.0 --port 7100

backend:
	cd backend && go run .

web:
	cd web && bun run dev

tidy:
	cd backend && go mod tidy
	cd web && bun install

test:
	cd backend && go test ./...
	cd web && bun run typecheck
	cd web && bun run build
	python3 -m py_compile scripts/clone_voice.py scripts/validate_detector.py scripts/build_audio_manifest.py scripts/extract_spoof_features.py scripts/train_spoof_meta.py scripts/modal_audio_oss_pipeline.py

manifest:
	python3 scripts/build_audio_manifest.py --adapter fixtures-user --root fixtures/user --out data/audio_manifest.csv

modal-volume:
	MODAL_PROFILE=$(MODAL_PROFILE) modal volume create $(MODAL_VOLUME) || true

modal-put-fixtures: manifest
	MODAL_PROFILE=$(MODAL_PROFILE) modal volume put --force $(MODAL_VOLUME) fixtures/user $(MODAL_PREFIX)/fixtures/user
	MODAL_PROFILE=$(MODAL_PROFILE) modal volume put --force $(MODAL_VOLUME) data/audio_manifest.csv $(MODAL_PREFIX)/manifests/audio_manifest.csv
	MODAL_PROFILE=$(MODAL_PROFILE) modal volume put --force $(MODAL_VOLUME) docs/detector_validation_results.csv $(MODAL_PREFIX)/reports/detector_validation_results.csv
	MODAL_PROFILE=$(MODAL_PROFILE) modal volume put --force $(MODAL_VOLUME) docs/spoof_features.csv $(MODAL_PREFIX)/reports/spoof_features.csv
	MODAL_PROFILE=$(MODAL_PROFILE) modal volume put --force $(MODAL_VOLUME) docs/SPOOF_META_RESULTS.md $(MODAL_PREFIX)/reports/SPOOF_META_RESULTS.md
	MODAL_PROFILE=$(MODAL_PROFILE) modal volume put --force $(MODAL_VOLUME) docs/spoof_features_manifest_smoke.csv $(MODAL_PREFIX)/reports/spoof_features_manifest_smoke.csv
	MODAL_PROFILE=$(MODAL_PROFILE) modal volume put --force $(MODAL_VOLUME) docs/SPOOF_META_MANIFEST_SMOKE.md $(MODAL_PREFIX)/reports/SPOOF_META_MANIFEST_SMOKE.md
	MODAL_PROFILE=$(MODAL_PROFILE) modal volume put --force $(MODAL_VOLUME) docs/SPOOF_META_MANIFEST_SPLIT_SMOKE.md $(MODAL_PREFIX)/reports/SPOOF_META_MANIFEST_SPLIT_SMOKE.md

modal-run-in-the-wild:
	MODAL_PROFILE=$(MODAL_PROFILE) modal run scripts/modal_audio_oss_pipeline.py --dataset in-the-wild --max-per-label 500

modal-ls:
	MODAL_PROFILE=$(MODAL_PROFILE) modal volume ls $(MODAL_VOLUME) $(MODAL_PREFIX)

reset-demo:
	rm -f data/enrollments.json data/audit.jsonl
	rm -f .secrets/server-ed25519.key .secrets/voiceprint-salt.key

clean:
	rm -rf web/dist web/node_modules
	rm -f web/tsconfig.tsbuildinfo
	find scripts -type d -name __pycache__ -prune -exec rm -rf {} +
	rm -f backend/backend backend/main
