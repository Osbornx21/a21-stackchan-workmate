PLATFORMIO_CORE_DIR := $(CURDIR)/.a21-tools/platformio-core
PIO := env PLATFORMIO_CORE_DIR="$(PLATFORMIO_CORE_DIR)" .a21-tools/platformio-venv/bin/pio
A21_DEVICE_MAX_AGE_MS ?= 300000
A21_GATEWAY_URL ?= http://127.0.0.1:21080
A21_LAN_SAMPLES ?= 5
A21_MIC_PROBE_MIN_ABS_PEAK ?= 1
A21_MIC_PROBE_MIN_FRAMES ?= 90
A21_MIC_PROBE_MIN_GATEWAY_RMS ?= 0
A21_MIC_PROBE_MIN_NONZERO_SAMPLES ?= 1
A21_MIC_PROBE_MIN_VAD_SPEECH ?= 0
A21_MIC_PROBE_MIN_DELIVERY_RATIO ?= 0.95
A21_MIC_PROBE_WINDOW_MS ?= 0
A21_HALF_DUPLEX_MIN_DELIVERY_RATIO ?= 0.95
A21_HALF_DUPLEX_MIN_MIC_FRAMES ?= 1
A21_HALF_DUPLEX_MIN_PLAYBACK_CHUNKS ?= 1
A21_HALF_DUPLEX_WINDOW_MS ?= 1500
A21_PLATFORMIO_VERSION ?= 6.1.19
A21_SPEAKER_MOCK_AUDIO_CHUNKS ?= 50
A21_SPEAKER_MIN_PLAYED_FRAMES ?= 50
A21_SPEAKER_WINDOW_MS ?= 1000

.PHONY: test verify preflight namespace-audit doctor gateway lan-probe provider-smoke provider-smoke-execute provider-realtime-plan provider-realtime-fixture v21-adapter-smoke v21-adapter-smoke-execute audio-front-end-eval latency-bench release-check firmware-tools firmware-check firmware-test firmware-build firmware-mic-probe-build firmware-avatar-spike-build firmware-upload-blocker-check firmware-mic-probe-upload-blocker-check firmware-clean-check firmware-package firmware-current-artifact-check firmware-artifact-prune-plan firmware-artifact-check firmware-upload-check firmware-device-report office-handoff office-preflight office-acceptance stackchan-identity-acceptance stackchan-physical-evidence stackchan-capability-acceptance stackchan-mic-probe-acceptance stackchan-half-duplex-acceptance stackchan-speaker-acceptance stackchan-touch-acceptance firmware-device-check firmware-flash-plan firmware-bootstrap-flash-plan firmware-bootstrap-flash-execute firmware-mic-probe-flash-plan firmware-mic-probe-flash-execute

test:
	go test ./...

preflight:
	go run ./cmd/a21 preflight

namespace-audit:
	go run ./cmd/a21 namespace-audit

doctor:
	go run ./cmd/a21 doctor

gateway:
	go run ./cmd/a21 gateway --addr 127.0.0.1:21080

lan-probe:
	@test -n "$(A21_LAN_TARGET)" || (echo "A21_LAN_TARGET is required, for example A21_LAN_TARGET=a21-gateway=127.0.0.1:21080"; exit 2)
	go run ./cmd/a21 lan-probe --target "$(A21_LAN_TARGET)" --samples "$(A21_LAN_SAMPLES)" --output-dir reports

provider-smoke:
	go run ./cmd/a21 provider-smoke

provider-smoke-execute:
	@test -n "$(A21_PROVIDER)" || (echo "A21_PROVIDER is required"; exit 2)
	go run ./cmd/a21 provider-smoke --provider "$(A21_PROVIDER)" --execute

provider-realtime-plan:
	@if [ -n "$(A21_PROVIDER)" ]; then \
		go run ./cmd/a21 provider-realtime-plan --provider "$(A21_PROVIDER)"; \
	else \
		go run ./cmd/a21 provider-realtime-plan; \
	fi

provider-realtime-fixture:
	@test -n "$(A21_PROVIDER)" || (echo "A21_PROVIDER is required"; exit 2)
	go run ./cmd/a21 provider-realtime-fixture --provider "$(A21_PROVIDER)" --execute

v21-adapter-smoke:
	go run ./cmd/a21 v21-adapter-smoke --output-dir reports

v21-adapter-smoke-execute:
	@test -n "$(A21_V21_ADAPTER_URL)" || (echo "A21_V21_ADAPTER_URL is required"; exit 2)
	go run ./cmd/a21 v21-adapter-smoke --execute --output-dir reports

audio-front-end-eval:
	@if [ -n "$(A21_AUDIO_FIXTURE)" ]; then \
		go run ./cmd/a21 audio-front-end-eval --fixture "$(A21_AUDIO_FIXTURE)" --output-dir reports; \
	else \
		go run ./cmd/a21 audio-front-end-eval --mock --output-dir reports; \
	fi

latency-bench:
	go run ./cmd/a21 latency-bench --mock --iterations 5 --output-dir reports

release-check: verify namespace-audit latency-bench firmware-test firmware-build firmware-upload-blocker-check firmware-package firmware-current-artifact-check firmware-artifact-prune-plan office-handoff doctor

firmware-tools:
	A21_PLATFORMIO_VERSION="$(A21_PLATFORMIO_VERSION)" scripts/a21_setup_platformio.sh

firmware-check:
	go run ./cmd/a21 firmware-check

firmware-test: firmware-tools firmware-check
	$(PIO) test -d firmware/stackchan -e a21_stackchan_native

firmware-build: firmware-tools firmware-check
	$(PIO) run -d firmware/stackchan

firmware-mic-probe-build: firmware-tools firmware-check
	$(PIO) run -d firmware/stackchan -e a21_stackchan_cores3_mic_probe

firmware-avatar-spike-build: firmware-tools firmware-check
	$(PIO) run -d firmware/stackchan -e a21_stackchan_cores3_avatar_spike

firmware-upload-blocker-check: firmware-tools
	@output="$$( $(PIO) run -d firmware/stackchan -e a21_stackchan_cores3 -t upload 2>&1 )"; \
	code="$$?"; \
	printf '%s\n' "$$output"; \
	if [ "$$code" -eq 0 ]; then \
		echo "A21 raw PlatformIO upload blocker failed: upload target exited 0"; \
		exit 1; \
	fi; \
	printf '%s\n' "$$output" | grep -q "A21 raw PlatformIO upload is forbidden" || { \
		echo "A21 raw PlatformIO upload blocker failed: guard message missing"; \
		exit 1; \
	}; \
	echo "A21 raw PlatformIO upload blocker ok"

firmware-mic-probe-upload-blocker-check: firmware-tools
	@output="$$( $(PIO) run -d firmware/stackchan -e a21_stackchan_cores3_mic_probe -t upload 2>&1 )"; \
	code="$$?"; \
	printf '%s\n' "$$output"; \
	if [ "$$code" -eq 0 ]; then \
		echo "A21 mic probe raw PlatformIO upload blocker failed: upload target exited 0"; \
		exit 1; \
	fi; \
	printf '%s\n' "$$output" | grep -q "A21 raw PlatformIO upload is forbidden" || { \
		echo "A21 mic probe raw PlatformIO upload blocker failed: guard message missing"; \
		exit 1; \
	}; \
	echo "A21 mic probe raw PlatformIO upload blocker ok"

firmware-clean-check:
	@test -z "$$(git status --porcelain --untracked-files=all)" || (echo "A21 firmware package requires a clean git worktree"; git status --short; exit 2)

firmware-package: firmware-clean-check firmware-build
	go run ./cmd/a21 firmware-package --commit $$(git rev-parse --short=12 HEAD)

firmware-current-artifact-check:
	go run ./cmd/a21 firmware-current-artifact-check --commit $$(git rev-parse --short=12 HEAD)

firmware-artifact-prune-plan:
	go run ./cmd/a21 firmware-artifact-prune-plan --commit $$(git rev-parse --short=12 HEAD) --output-dir reports

firmware-artifact-check:
	@test -n "$(A21_FIRMWARE_ARTIFACT)" || (echo "A21_FIRMWARE_ARTIFACT is required"; exit 2)
	go run ./cmd/a21 firmware-artifact-check --artifact "$(A21_FIRMWARE_ARTIFACT)"

firmware-upload-check:
	@test -n "$(A21_FIRMWARE_ARTIFACT)" || (echo "A21_FIRMWARE_ARTIFACT is required"; exit 2)
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	go run ./cmd/a21 firmware-upload-check --artifact "$(A21_FIRMWARE_ARTIFACT)" --port "$(A21_UPLOAD_PORT)" --commit $$(git rev-parse --short=12 HEAD)

firmware-device-report:
	go run ./cmd/a21 firmware-device-report --gateway-url "$(A21_GATEWAY_URL)" --output-dir reports

office-handoff:
	go run ./cmd/a21 office-handoff --commit $$(git rev-parse --short=12 HEAD) --output-dir reports

office-preflight:
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	go run ./cmd/a21 office-preflight --gateway-url "$(A21_GATEWAY_URL)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --max-device-age-ms "$(A21_DEVICE_MAX_AGE_MS)" --output-dir reports

office-acceptance:
	@test -n "$(A21_HANDOFF_REPORT)" || (echo "A21_HANDOFF_REPORT is required"; exit 2)
	@test -n "$(A21_OFFICE_PREFLIGHT_REPORT)" || (echo "A21_OFFICE_PREFLIGHT_REPORT is required"; exit 2)
	@if [ -n "$(A21_FIRMWARE_FLASH_PLAN)" ]; then \
		go run ./cmd/a21 office-acceptance --handoff "$(A21_HANDOFF_REPORT)" --office-preflight "$(A21_OFFICE_PREFLIGHT_REPORT)" --firmware-flash-plan "$(A21_FIRMWARE_FLASH_PLAN)" --output-dir reports; \
	else \
		go run ./cmd/a21 office-acceptance --handoff "$(A21_HANDOFF_REPORT)" --office-preflight "$(A21_OFFICE_PREFLIGHT_REPORT)" --output-dir reports; \
	fi

stackchan-identity-acceptance:
	@test -n "$(A21_OFFICE_ACCEPTANCE_REPORT)" || (echo "A21_OFFICE_ACCEPTANCE_REPORT is required"; exit 2)
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	go run ./cmd/a21 stackchan-identity-acceptance --office-acceptance "$(A21_OFFICE_ACCEPTANCE_REPORT)" --gateway-url "$(A21_GATEWAY_URL)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --max-device-age-ms "$(A21_DEVICE_MAX_AGE_MS)" --output-dir reports

stackchan-physical-evidence:
	@test -n "$(A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT)" || (echo "A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT is required"; exit 2)
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	@if [ "$(A21_DERIVE_GATEWAY_EVIDENCE)" = "1" ]; then \
		go run ./cmd/a21 stackchan-physical-evidence --identity-acceptance "$(A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --derive-gateway --gateway-url "$(A21_GATEWAY_URL)" --output-dir reports; \
	else \
		go run ./cmd/a21 stackchan-physical-evidence --identity-acceptance "$(A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --output-dir reports; \
	fi

stackchan-capability-acceptance:
	@test -n "$(A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT)" || (echo "A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT is required"; exit 2)
	@test -n "$(A21_STACKCHAN_PHYSICAL_EVIDENCE_REPORT)" || (echo "A21_STACKCHAN_PHYSICAL_EVIDENCE_REPORT is required"; exit 2)
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	go run ./cmd/a21 stackchan-capability-acceptance --identity-acceptance "$(A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT)" --evidence "$(A21_STACKCHAN_PHYSICAL_EVIDENCE_REPORT)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --output-dir reports

stackchan-mic-probe-acceptance:
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	go run ./cmd/a21 stackchan-mic-probe-acceptance --gateway-url "$(A21_GATEWAY_URL)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --window-ms "$(A21_MIC_PROBE_WINDOW_MS)" --min-frames "$(A21_MIC_PROBE_MIN_FRAMES)" --min-abs-peak "$(A21_MIC_PROBE_MIN_ABS_PEAK)" --min-nonzero-samples "$(A21_MIC_PROBE_MIN_NONZERO_SAMPLES)" --min-gateway-rms "$(A21_MIC_PROBE_MIN_GATEWAY_RMS)" --min-vad-speech "$(A21_MIC_PROBE_MIN_VAD_SPEECH)" --min-delivery-ratio "$(A21_MIC_PROBE_MIN_DELIVERY_RATIO)" --output-dir reports

stackchan-half-duplex-acceptance:
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	go run ./cmd/a21 stackchan-half-duplex-acceptance --gateway-url "$(A21_GATEWAY_URL)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --window-ms "$(A21_HALF_DUPLEX_WINDOW_MS)" --min-mic-frames "$(A21_HALF_DUPLEX_MIN_MIC_FRAMES)" --min-playback-chunks "$(A21_HALF_DUPLEX_MIN_PLAYBACK_CHUNKS)" --min-delivery-ratio "$(A21_HALF_DUPLEX_MIN_DELIVERY_RATIO)" --output-dir reports

stackchan-speaker-acceptance:
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	go run ./cmd/a21 stackchan-speaker-acceptance --gateway-url "$(A21_GATEWAY_URL)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --window-ms "$(A21_SPEAKER_WINDOW_MS)" --mock-audio-chunks "$(A21_SPEAKER_MOCK_AUDIO_CHUNKS)" --min-played-frames "$(A21_SPEAKER_MIN_PLAYED_FRAMES)" --output-dir reports

stackchan-touch-acceptance:
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	@test -n "$(A21_TOUCH_CASE)" || (echo "A21_TOUCH_CASE is required"; exit 2)
	go run ./cmd/a21 stackchan-touch-acceptance --gateway-url "$(A21_GATEWAY_URL)" --device-id "$(A21_DEVICE_ID)" --case "$(A21_TOUCH_CASE)" --window-ms "$${A21_TOUCH_WINDOW_MS:-15000}" --output-dir reports

firmware-device-check:
	@test -n "$(A21_FIRMWARE_ARTIFACT)" || (echo "A21_FIRMWARE_ARTIFACT is required"; exit 2)
	@test -n "$(A21_DEVICE_REPORT)" || (echo "A21_DEVICE_REPORT is required"; exit 2)
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	go run ./cmd/a21 firmware-device-check --artifact "$(A21_FIRMWARE_ARTIFACT)" --device-report "$(A21_DEVICE_REPORT)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --max-device-age-ms "$(A21_DEVICE_MAX_AGE_MS)"

firmware-flash-plan:
	@test -n "$(A21_FIRMWARE_ARTIFACT)" || (echo "A21_FIRMWARE_ARTIFACT is required"; exit 2)
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	@test -n "$(A21_DEVICE_REPORT)" || (echo "A21_DEVICE_REPORT is required"; exit 2)
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	go run ./cmd/a21 firmware-flash-plan --artifact "$(A21_FIRMWARE_ARTIFACT)" --port "$(A21_UPLOAD_PORT)" --device-report "$(A21_DEVICE_REPORT)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --max-device-age-ms "$(A21_DEVICE_MAX_AGE_MS)" --output-dir reports

firmware-bootstrap-flash-plan:
	@test -n "$(A21_FIRMWARE_ARTIFACT)" || (echo "A21_FIRMWARE_ARTIFACT is required"; exit 2)
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	go run ./cmd/a21 firmware-bootstrap-flash-plan --artifact "$(A21_FIRMWARE_ARTIFACT)" --port "$(A21_UPLOAD_PORT)" --commit $$(git rev-parse --short=12 HEAD) --output-dir reports

firmware-bootstrap-flash-execute:
	@test -n "$(A21_FIRMWARE_ARTIFACT)" || (echo "A21_FIRMWARE_ARTIFACT is required"; exit 2)
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	@test "$(A21_BOOTSTRAP_FLASH_CONFIRM)" = "WRITE_A21_STACKCHAN_FIRMWARE" || (echo "A21_BOOTSTRAP_FLASH_CONFIRM=WRITE_A21_STACKCHAN_FIRMWARE is required"; exit 2)
	go run ./cmd/a21 firmware-bootstrap-flash-execute --artifact "$(A21_FIRMWARE_ARTIFACT)" --port "$(A21_UPLOAD_PORT)" --commit $$(git rev-parse --short=12 HEAD) --confirm "$(A21_BOOTSTRAP_FLASH_CONFIRM)" --output-dir reports

firmware-mic-probe-flash-plan: firmware-mic-probe-build
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	go run ./cmd/a21 firmware-mic-probe-flash-plan --port "$(A21_UPLOAD_PORT)" --commit $$(git rev-parse --short=12 HEAD) --output-dir reports

firmware-mic-probe-flash-execute: firmware-mic-probe-build
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	@test "$(A21_MIC_PROBE_FLASH_CONFIRM)" = "WRITE_A21_STACKCHAN_MIC_PROBE_FIRMWARE" || (echo "A21_MIC_PROBE_FLASH_CONFIRM=WRITE_A21_STACKCHAN_MIC_PROBE_FIRMWARE is required"; exit 2)
	go run ./cmd/a21 firmware-mic-probe-flash-execute --port "$(A21_UPLOAD_PORT)" --commit $$(git rev-parse --short=12 HEAD) --confirm "$(A21_MIC_PROBE_FLASH_CONFIRM)" --output-dir reports

verify:
	go test ./...
	git diff --check
