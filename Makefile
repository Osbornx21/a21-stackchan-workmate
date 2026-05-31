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
A21_IMU_PROBE_MAX_READ_ERRORS ?= 0
A21_IMU_PROBE_MIN_ACCEL_TOTAL_MG ?= 500
A21_IMU_PROBE_MIN_SAMPLES ?= 10
A21_IMU_PROBE_WINDOW_MS ?= 1500
A21_SENSOR_PROBE_MAX_READ_ERRORS ?= 0
A21_SENSOR_PROBE_MIN_BATTERY_MV ?= 3000
A21_SENSOR_PROBE_MIN_SAMPLES ?= 10
A21_SENSOR_PROBE_WINDOW_MS ?= 1500
A21_HALF_DUPLEX_MIN_DELIVERY_RATIO ?= 0.95
A21_HALF_DUPLEX_MIN_MIC_FRAMES ?= 1
A21_HALF_DUPLEX_MIN_PLAYBACK_CHUNKS ?= 1
A21_HALF_DUPLEX_WINDOW_MS ?= 1500
A21_PLATFORMIO_VERSION ?= 6.1.19
A21_SPEAKER_MOCK_AUDIO_CHUNKS ?= 50
A21_SPEAKER_MIN_PLAYED_FRAMES ?= 50
A21_SPEAKER_WINDOW_MS ?= 1500
A21_STACKCHAN_OFFICIAL_SOURCE ?= /Users/jiyurun/Documents/小马暴力/sources/m5stack-stackchan
A21_STACKCHAN_OFFICIAL_WORK_DIR ?= /tmp/a21-stackchan-official-clean
A21_STACKCHAN_OFFICIAL_BUILD_DIR ?= /tmp/a21-stackchan-official-build
A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE_OVERLAY ?= firmware/stackchan-official/overlays/a21-official-audio-smoke.patch
A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_OVERLAY ?= firmware/stackchan-official/overlays/a21-official-pcm-bridge.patch
A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL ?=
A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE_FLASH_CONFIRM ?=
A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS_CONFIRM ?=
A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_APP_FLASH_CONFIRM ?=
A21_CONTROL_COMMAND ?=
A21_CONTROL_TIER ?=
A21_IDF_EXPORT ?= /Users/jiyurun/esp/esp-idf-v5.5.2/export.sh
A21_V21_ADAPTER_ADDR ?= 127.0.0.1:21121
A21_V21_BACKEND_URL ?= http://127.0.0.1:18080

.PHONY: test verify gate preflight namespace-audit promotion-readiness control-guard doctor gateway demo product-readiness agent-plan agent-io-smoke agent-io-smoke-execute lan-probe provider-smoke provider-smoke-execute provider-realtime-plan provider-realtime-fixture provider-latency-bench v21-adapter-bridge v21-adapter-smoke v21-adapter-smoke-execute audio-front-end-eval local-tts-smoke local-asr-smoke local-voice-loopback stackchan-local-tts-playback stackchan-fast-companion-turn stackchan-official-baseline stackchan-official-baseline-build stackchan-official-audio-smoke-build stackchan-official-pcm-bridge-build stackchan-official-audio-smoke-flash-plan stackchan-official-audio-smoke-flash-execute stackchan-official-pcm-bridge-flash-plan stackchan-official-pcm-bridge-nvs-plan stackchan-official-pcm-bridge-nvs-execute latency-bench release-check firmware-tools firmware-check firmware-test firmware-build firmware-mic-probe-build firmware-imu-probe-build firmware-sensor-probe-build firmware-avatar-spike-build firmware-upload-blocker-check firmware-mic-probe-upload-blocker-check firmware-imu-probe-upload-blocker-check firmware-sensor-probe-upload-blocker-check firmware-clean-check firmware-package firmware-current-artifact-check firmware-artifact-prune-plan firmware-artifact-check firmware-upload-check firmware-device-report office-handoff office-preflight office-acceptance stackchan-identity-acceptance stackchan-physical-evidence stackchan-capability-acceptance stackchan-mic-probe-acceptance stackchan-imu-probe-acceptance stackchan-sensor-probe-acceptance stackchan-half-duplex-acceptance stackchan-speaker-acceptance stackchan-touch-acceptance stackchan-hardware-mainline firmware-device-check firmware-flash-plan firmware-bootstrap-flash-plan firmware-bootstrap-flash-execute firmware-mic-probe-flash-plan firmware-mic-probe-flash-execute firmware-imu-probe-flash-plan firmware-imu-probe-flash-execute firmware-sensor-probe-flash-plan firmware-sensor-probe-flash-execute

test:
	go test ./...

gate:
	go run ./cmd/a21 gate --scope host

preflight:
	go run ./cmd/a21 gate --scope host

namespace-audit:
	go run ./cmd/a21 gate --scope host

promotion-readiness:
	go run ./cmd/a21 gate --scope integration

control-guard:
	@test -n "$(A21_CONTROL_COMMAND)" || (echo "A21_CONTROL_COMMAND is required, for example A21_CONTROL_COMMAND='provider-smoke --execute'"; exit 2)
	@if [ -n "$(A21_CONTROL_TIER)" ]; then \
		go run ./cmd/a21 gate --scope hardware --command "$(A21_CONTROL_COMMAND)" --tier "$(A21_CONTROL_TIER)"; \
	else \
		go run ./cmd/a21 gate --scope hardware --command "$(A21_CONTROL_COMMAND)"; \
	fi

doctor:
	go run ./cmd/a21 gate --scope host

gateway:
	go run ./cmd/a21 gateway --addr 127.0.0.1:21080

demo:
	go run ./cmd/a21 demo --addr 127.0.0.1:21080 --open

product-readiness:
	go run ./cmd/a21 product-readiness --gateway-url "$(A21_GATEWAY_URL)" --device-id "$${A21_DEVICE_ID:-stackchan-001}" --output-dir reports

agent-plan:
	go run ./cmd/a21 agent-plan --output-dir reports

agent-io-smoke:
	go run ./cmd/a21 agent-io-smoke --output-dir reports

agent-io-smoke-execute:
	@test -n "$(A21_HERMES_AGENT_URL)$(A21_AGENT_IO_ENDPOINT_URL)" || (echo "A21_HERMES_AGENT_URL or A21_AGENT_IO_ENDPOINT_URL is required"; exit 2)
	@test -n "$(A21_HERMES_AGENT_KEY)$(A21_AGENT_IO_API_KEY)" || (echo "A21_HERMES_AGENT_KEY or A21_AGENT_IO_API_KEY is required"; exit 2)
	go run ./cmd/a21 agent-io-smoke --execute --output-dir reports

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

provider-latency-bench:
	go run ./cmd/a21 provider-latency-bench --provider "$${A21_PROVIDER:-mock}" --iterations "$${A21_PROVIDER_LATENCY_ITERATIONS:-5}" --output-dir reports

v21-adapter-bridge:
	go run ./cmd/a21 v21-adapter-bridge --addr "$(A21_V21_ADAPTER_ADDR)" --v21-url "$(A21_V21_BACKEND_URL)"

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

local-tts-smoke:
	go run ./cmd/a21 local-tts-smoke --output-dir reports

local-asr-smoke:
	go run ./cmd/a21 local-asr-smoke --output-dir reports

local-voice-loopback:
	go run ./cmd/a21 local-voice-loopback --repeat 3 --output-dir reports

stackchan-local-tts-playback:
	go run ./cmd/a21 stackchan-local-tts-playback --output-dir reports

stackchan-fast-companion-turn:
	go run ./cmd/a21 stackchan-fast-companion-turn --repeat 3 --output-dir reports

stackchan-official-baseline:
	go run ./cmd/a21 stackchan-official-baseline --source "$(A21_STACKCHAN_OFFICIAL_SOURCE)" --work-dir "$(A21_STACKCHAN_OFFICIAL_WORK_DIR)" --build-dir "$(A21_STACKCHAN_OFFICIAL_BUILD_DIR)" --idf-export "$(A21_IDF_EXPORT)" --output-dir reports

stackchan-official-baseline-build:
	go run ./cmd/a21 stackchan-official-baseline --source "$(A21_STACKCHAN_OFFICIAL_SOURCE)" --work-dir "$(A21_STACKCHAN_OFFICIAL_WORK_DIR)" --build-dir "$(A21_STACKCHAN_OFFICIAL_BUILD_DIR)" --idf-export "$(A21_IDF_EXPORT)" --execute --output-dir reports

stackchan-official-audio-smoke-build:
	go run ./cmd/a21 stackchan-official-baseline --source "$(A21_STACKCHAN_OFFICIAL_SOURCE)" --work-dir "$(A21_STACKCHAN_OFFICIAL_WORK_DIR)" --build-dir "$(A21_STACKCHAN_OFFICIAL_BUILD_DIR)" --idf-export "$(A21_IDF_EXPORT)" --overlay "$(A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE_OVERLAY)" --execute --output-dir reports

stackchan-official-pcm-bridge-build:
	go run ./cmd/a21 stackchan-official-baseline --source "$(A21_STACKCHAN_OFFICIAL_SOURCE)" --work-dir "$(A21_STACKCHAN_OFFICIAL_WORK_DIR)" --build-dir "$(A21_STACKCHAN_OFFICIAL_BUILD_DIR)" --idf-export "$(A21_IDF_EXPORT)" --overlay "$(A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_OVERLAY)" --execute --output-dir reports

stackchan-official-audio-smoke-flash-plan:
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	go run ./cmd/a21 stackchan-official-audio-smoke-flash --build-dir "$(A21_STACKCHAN_OFFICIAL_BUILD_DIR)" --idf-export "$(A21_IDF_EXPORT)" --port "$(A21_UPLOAD_PORT)" --output-dir reports

stackchan-official-audio-smoke-flash-execute:
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	@test -n "$(A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE_FLASH_CONFIRM)" || (echo "A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE_FLASH_CONFIRM is required"; exit 2)
	go run ./cmd/a21 stackchan-official-audio-smoke-flash --execute --build-dir "$(A21_STACKCHAN_OFFICIAL_BUILD_DIR)" --idf-export "$(A21_IDF_EXPORT)" --port "$(A21_UPLOAD_PORT)" --confirm "$(A21_STACKCHAN_OFFICIAL_AUDIO_SMOKE_FLASH_CONFIRM)" --output-dir reports

stackchan-official-pcm-bridge-flash-plan:
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	@test -n "$(A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL)" || (echo "A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL is required"; exit 2)
	go run ./cmd/a21 stackchan-official-pcm-bridge-flash --build-dir "$(A21_STACKCHAN_OFFICIAL_BUILD_DIR)" --idf-export "$(A21_IDF_EXPORT)" --port "$(A21_UPLOAD_PORT)" --device-id "$${A21_DEVICE_ID:-stackchan-001}" --audio-ws-url "$(A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL)" --output-dir reports

stackchan-official-pcm-bridge-flash-execute:
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	@test -n "$(A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL)" || (echo "A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL is required"; exit 2)
	@test -n "$(A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_APP_FLASH_CONFIRM)" || (echo "A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_APP_FLASH_CONFIRM is required"; exit 2)
	go run ./cmd/a21 stackchan-official-pcm-bridge-flash --execute --build-dir "$(A21_STACKCHAN_OFFICIAL_BUILD_DIR)" --idf-export "$(A21_IDF_EXPORT)" --port "$(A21_UPLOAD_PORT)" --device-id "$${A21_DEVICE_ID:-stackchan-001}" --audio-ws-url "$(A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL)" --confirm "$(A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_APP_FLASH_CONFIRM)" --output-dir reports

stackchan-official-pcm-bridge-nvs-plan:
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	@test -n "$(A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL)" || (echo "A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL is required"; exit 2)
	go run ./cmd/a21 stackchan-official-pcm-bridge-nvs --idf-export "$(A21_IDF_EXPORT)" --port "$(A21_UPLOAD_PORT)" --device-id "$${A21_DEVICE_ID:-stackchan-001}" --audio-ws-url "$(A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL)" --output-dir reports

stackchan-official-pcm-bridge-nvs-execute:
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	@test -n "$(A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL)" || (echo "A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL is required"; exit 2)
	@test -n "$(A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS_CONFIRM)" || (echo "A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS_CONFIRM is required"; exit 2)
	go run ./cmd/a21 stackchan-official-pcm-bridge-nvs --execute --idf-export "$(A21_IDF_EXPORT)" --port "$(A21_UPLOAD_PORT)" --device-id "$${A21_DEVICE_ID:-stackchan-001}" --audio-ws-url "$(A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_AUDIO_WS_URL)" --confirm "$(A21_STACKCHAN_OFFICIAL_PCM_BRIDGE_NVS_CONFIRM)" --output-dir reports

latency-bench:
	go run ./cmd/a21 latency-bench --mock --iterations 5 --output-dir reports

release-check: verify gate latency-bench firmware-test firmware-build firmware-upload-blocker-check firmware-mic-probe-upload-blocker-check firmware-imu-probe-upload-blocker-check firmware-package firmware-current-artifact-check firmware-artifact-prune-plan office-handoff

firmware-tools:
	A21_PLATFORMIO_VERSION="$(A21_PLATFORMIO_VERSION)" scripts/a21_setup_platformio.sh

firmware-check:
	go run ./cmd/a21 firmware-check --kind manifest

firmware-test: firmware-tools firmware-check
	$(PIO) test -d firmware/stackchan -e a21_stackchan_native

firmware-build: firmware-tools firmware-check
	$(PIO) run -d firmware/stackchan

firmware-mic-probe-build: firmware-tools firmware-check
	$(PIO) run -d firmware/stackchan -e a21_stackchan_cores3_mic_probe

firmware-imu-probe-build: firmware-tools firmware-check
	$(PIO) run -d firmware/stackchan -e a21_stackchan_cores3_imu_probe

firmware-sensor-probe-build: firmware-tools firmware-check
	$(PIO) run -d firmware/stackchan -e a21_stackchan_cores3_sensor_probe

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

firmware-imu-probe-upload-blocker-check: firmware-tools
	@output="$$( $(PIO) run -d firmware/stackchan -e a21_stackchan_cores3_imu_probe -t upload 2>&1 )"; \
	code="$$?"; \
	printf '%s\n' "$$output"; \
	if [ "$$code" -eq 0 ]; then \
		echo "A21 IMU probe raw PlatformIO upload blocker failed: upload target exited 0"; \
		exit 1; \
	fi; \
	printf '%s\n' "$$output" | grep -q "A21 raw PlatformIO upload is forbidden" || { \
		echo "A21 IMU probe raw PlatformIO upload blocker failed: guard message missing"; \
		exit 1; \
	}; \
	echo "A21 IMU probe raw PlatformIO upload blocker ok"

firmware-sensor-probe-upload-blocker-check: firmware-tools
	@output="$$( $(PIO) run -d firmware/stackchan -e a21_stackchan_cores3_sensor_probe -t upload 2>&1 )"; \
	code="$$?"; \
	printf '%s\n' "$$output"; \
	if [ "$$code" -eq 0 ]; then \
		echo "A21 sensor probe raw PlatformIO upload blocker failed: upload target exited 0"; \
		exit 1; \
	fi; \
	printf '%s\n' "$$output" | grep -q "A21 raw PlatformIO upload is forbidden" || { \
		echo "A21 sensor probe raw PlatformIO upload blocker failed: guard message missing"; \
		exit 1; \
	}; \
	echo "A21 sensor probe raw PlatformIO upload blocker ok"

firmware-clean-check:
	@test -z "$$(git status --porcelain --untracked-files=all)" || (echo "A21 firmware package requires a clean git worktree"; git status --short; exit 2)

firmware-package: firmware-clean-check firmware-build
	go run ./cmd/a21 firmware-package --commit $$(git rev-parse --short=12 HEAD)

firmware-current-artifact-check:
	go run ./cmd/a21 firmware-check --kind current-artifact --commit $$(git rev-parse --short=12 HEAD)

firmware-artifact-prune-plan:
	go run ./cmd/a21 firmware-artifact-prune-plan --commit $$(git rev-parse --short=12 HEAD) --output-dir reports

firmware-artifact-check:
	@test -n "$(A21_FIRMWARE_ARTIFACT)" || (echo "A21_FIRMWARE_ARTIFACT is required"; exit 2)
	go run ./cmd/a21 firmware-check --kind artifact --artifact "$(A21_FIRMWARE_ARTIFACT)"

firmware-upload-check:
	@test -n "$(A21_FIRMWARE_ARTIFACT)" || (echo "A21_FIRMWARE_ARTIFACT is required"; exit 2)
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	go run ./cmd/a21 firmware-check --kind upload --artifact "$(A21_FIRMWARE_ARTIFACT)" --port "$(A21_UPLOAD_PORT)" --commit $$(git rev-parse --short=12 HEAD)

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
	go run ./cmd/a21 stackchan-accept --check identity --office-acceptance "$(A21_OFFICE_ACCEPTANCE_REPORT)" --gateway-url "$(A21_GATEWAY_URL)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --max-device-age-ms "$(A21_DEVICE_MAX_AGE_MS)" --output-dir reports

stackchan-physical-evidence:
	@test -n "$(A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT)" || (echo "A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT is required"; exit 2)
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	@if [ "$(A21_DERIVE_GATEWAY_EVIDENCE)" = "1" ]; then \
		go run ./cmd/a21 stackchan-accept --check physical-evidence --identity-acceptance "$(A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --derive-gateway --gateway-url "$(A21_GATEWAY_URL)" --output-dir reports; \
	else \
		go run ./cmd/a21 stackchan-accept --check physical-evidence --identity-acceptance "$(A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --output-dir reports; \
	fi

stackchan-capability-acceptance:
	@test -n "$(A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT)" || (echo "A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT is required"; exit 2)
	@test -n "$(A21_STACKCHAN_PHYSICAL_EVIDENCE_REPORT)" || (echo "A21_STACKCHAN_PHYSICAL_EVIDENCE_REPORT is required"; exit 2)
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	go run ./cmd/a21 stackchan-accept --check capability --identity-acceptance "$(A21_STACKCHAN_IDENTITY_ACCEPTANCE_REPORT)" --evidence "$(A21_STACKCHAN_PHYSICAL_EVIDENCE_REPORT)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --output-dir reports

stackchan-mic-probe-acceptance:
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	go run ./cmd/a21 stackchan-accept --check mic-probe --gateway-url "$(A21_GATEWAY_URL)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --window-ms "$(A21_MIC_PROBE_WINDOW_MS)" --min-frames "$(A21_MIC_PROBE_MIN_FRAMES)" --min-abs-peak "$(A21_MIC_PROBE_MIN_ABS_PEAK)" --min-nonzero-samples "$(A21_MIC_PROBE_MIN_NONZERO_SAMPLES)" --min-gateway-rms "$(A21_MIC_PROBE_MIN_GATEWAY_RMS)" --min-vad-speech "$(A21_MIC_PROBE_MIN_VAD_SPEECH)" --min-delivery-ratio "$(A21_MIC_PROBE_MIN_DELIVERY_RATIO)" --output-dir reports

stackchan-imu-probe-acceptance:
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	go run ./cmd/a21 stackchan-accept --check imu-probe --gateway-url "$(A21_GATEWAY_URL)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --window-ms "$(A21_IMU_PROBE_WINDOW_MS)" --min-samples "$(A21_IMU_PROBE_MIN_SAMPLES)" --min-accel-total-mg "$(A21_IMU_PROBE_MIN_ACCEL_TOTAL_MG)" --max-read-errors "$(A21_IMU_PROBE_MAX_READ_ERRORS)" --output-dir reports

stackchan-sensor-probe-acceptance:
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	go run ./cmd/a21 stackchan-accept --check sensor-probe --gateway-url "$(A21_GATEWAY_URL)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --window-ms "$(A21_SENSOR_PROBE_WINDOW_MS)" --min-samples "$(A21_SENSOR_PROBE_MIN_SAMPLES)" --min-battery-mv "$(A21_SENSOR_PROBE_MIN_BATTERY_MV)" --max-read-errors "$(A21_SENSOR_PROBE_MAX_READ_ERRORS)" --output-dir reports

stackchan-half-duplex-acceptance:
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	go run ./cmd/a21 stackchan-accept --check half-duplex --gateway-url "$(A21_GATEWAY_URL)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --window-ms "$(A21_HALF_DUPLEX_WINDOW_MS)" --min-mic-frames "$(A21_HALF_DUPLEX_MIN_MIC_FRAMES)" --min-playback-chunks "$(A21_HALF_DUPLEX_MIN_PLAYBACK_CHUNKS)" --min-delivery-ratio "$(A21_HALF_DUPLEX_MIN_DELIVERY_RATIO)" --output-dir reports

stackchan-speaker-acceptance:
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	go run ./cmd/a21 stackchan-accept --check speaker --gateway-url "$(A21_GATEWAY_URL)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --window-ms "$(A21_SPEAKER_WINDOW_MS)" --mock-audio-chunks "$(A21_SPEAKER_MOCK_AUDIO_CHUNKS)" --min-played-frames "$(A21_SPEAKER_MIN_PLAYED_FRAMES)" --output-dir reports

stackchan-touch-acceptance:
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	@test -n "$(A21_TOUCH_CASE)" || (echo "A21_TOUCH_CASE is required"; exit 2)
	go run ./cmd/a21 stackchan-accept --check touch --gateway-url "$(A21_GATEWAY_URL)" --device-id "$(A21_DEVICE_ID)" --case "$(A21_TOUCH_CASE)" --window-ms "$${A21_TOUCH_WINDOW_MS:-15000}" --output-dir reports

stackchan-hardware-mainline:
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	go run ./cmd/a21 stackchan-accept --check hardware-mainline --gateway-url "$(A21_GATEWAY_URL)" --device-id "$(A21_DEVICE_ID)" --output-dir reports

firmware-device-check:
	@test -n "$(A21_FIRMWARE_ARTIFACT)" || (echo "A21_FIRMWARE_ARTIFACT is required"; exit 2)
	@test -n "$(A21_DEVICE_REPORT)" || (echo "A21_DEVICE_REPORT is required"; exit 2)
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	go run ./cmd/a21 firmware-check --kind device --artifact "$(A21_FIRMWARE_ARTIFACT)" --device-report "$(A21_DEVICE_REPORT)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --max-device-age-ms "$(A21_DEVICE_MAX_AGE_MS)"

firmware-flash-plan:
	@test -n "$(A21_FIRMWARE_ARTIFACT)" || (echo "A21_FIRMWARE_ARTIFACT is required"; exit 2)
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	@test -n "$(A21_DEVICE_REPORT)" || (echo "A21_DEVICE_REPORT is required"; exit 2)
	@test -n "$(A21_DEVICE_ID)" || (echo "A21_DEVICE_ID is required"; exit 2)
	go run ./cmd/a21 firmware-flash-plan --artifact "$(A21_FIRMWARE_ARTIFACT)" --port "$(A21_UPLOAD_PORT)" --device-report "$(A21_DEVICE_REPORT)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --max-device-age-ms "$(A21_DEVICE_MAX_AGE_MS)" --output-dir reports

firmware-bootstrap-flash-plan:
	@test -n "$(A21_FIRMWARE_ARTIFACT)" || (echo "A21_FIRMWARE_ARTIFACT is required"; exit 2)
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	go run ./cmd/a21 firmware-bootstrap-flash --artifact "$(A21_FIRMWARE_ARTIFACT)" --port "$(A21_UPLOAD_PORT)" --commit $$(git rev-parse --short=12 HEAD) --output-dir reports

firmware-bootstrap-flash-execute:
	@test -n "$(A21_FIRMWARE_ARTIFACT)" || (echo "A21_FIRMWARE_ARTIFACT is required"; exit 2)
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	@test "$(A21_BOOTSTRAP_FLASH_CONFIRM)" = "WRITE_A21_STACKCHAN_FIRMWARE" || (echo "A21_BOOTSTRAP_FLASH_CONFIRM=WRITE_A21_STACKCHAN_FIRMWARE is required"; exit 2)
	go run ./cmd/a21 firmware-bootstrap-flash --execute --artifact "$(A21_FIRMWARE_ARTIFACT)" --port "$(A21_UPLOAD_PORT)" --commit $$(git rev-parse --short=12 HEAD) --confirm "$(A21_BOOTSTRAP_FLASH_CONFIRM)" --output-dir reports

firmware-mic-probe-flash-plan: firmware-mic-probe-build
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	go run ./cmd/a21 firmware-mic-probe-flash --port "$(A21_UPLOAD_PORT)" --commit $$(git rev-parse --short=12 HEAD) --output-dir reports

firmware-mic-probe-flash-execute: firmware-mic-probe-build
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	@test "$(A21_MIC_PROBE_FLASH_CONFIRM)" = "WRITE_A21_STACKCHAN_MIC_PROBE_FIRMWARE" || (echo "A21_MIC_PROBE_FLASH_CONFIRM=WRITE_A21_STACKCHAN_MIC_PROBE_FIRMWARE is required"; exit 2)
	go run ./cmd/a21 firmware-mic-probe-flash --execute --port "$(A21_UPLOAD_PORT)" --commit $$(git rev-parse --short=12 HEAD) --confirm "$(A21_MIC_PROBE_FLASH_CONFIRM)" --output-dir reports

firmware-imu-probe-flash-plan: firmware-imu-probe-build
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	go run ./cmd/a21 firmware-imu-probe-flash --port "$(A21_UPLOAD_PORT)" --commit $$(git rev-parse --short=12 HEAD) --output-dir reports

firmware-imu-probe-flash-execute: firmware-imu-probe-build
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	@test "$(A21_IMU_PROBE_FLASH_CONFIRM)" = "WRITE_A21_STACKCHAN_IMU_PROBE_FIRMWARE" || (echo "A21_IMU_PROBE_FLASH_CONFIRM=WRITE_A21_STACKCHAN_IMU_PROBE_FIRMWARE is required"; exit 2)
	go run ./cmd/a21 firmware-imu-probe-flash --execute --port "$(A21_UPLOAD_PORT)" --commit $$(git rev-parse --short=12 HEAD) --confirm "$(A21_IMU_PROBE_FLASH_CONFIRM)" --output-dir reports

firmware-sensor-probe-flash-plan: firmware-sensor-probe-build
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	go run ./cmd/a21 firmware-sensor-probe-flash --port "$(A21_UPLOAD_PORT)" --commit $$(git rev-parse --short=12 HEAD) --output-dir reports

firmware-sensor-probe-flash-execute: firmware-sensor-probe-build
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	@test "$(A21_SENSOR_PROBE_FLASH_CONFIRM)" = "WRITE_A21_STACKCHAN_SENSOR_PROBE_FIRMWARE" || (echo "A21_SENSOR_PROBE_FLASH_CONFIRM=WRITE_A21_STACKCHAN_SENSOR_PROBE_FIRMWARE is required"; exit 2)
	go run ./cmd/a21 firmware-sensor-probe-flash --execute --port "$(A21_UPLOAD_PORT)" --commit $$(git rev-parse --short=12 HEAD) --confirm "$(A21_SENSOR_PROBE_FLASH_CONFIRM)" --output-dir reports

verify:
	go test ./...
	git diff --check
