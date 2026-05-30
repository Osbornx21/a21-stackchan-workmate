PLATFORMIO_CORE_DIR := $(CURDIR)/.a21-tools/platformio-core
PIO := env PLATFORMIO_CORE_DIR="$(PLATFORMIO_CORE_DIR)" .a21-tools/platformio-venv/bin/pio
A21_DEVICE_MAX_AGE_MS ?= 300000
A21_GATEWAY_URL ?= http://127.0.0.1:21080

.PHONY: test verify preflight doctor gateway provider-smoke provider-smoke-execute provider-realtime-plan provider-realtime-fixture audio-front-end-eval latency-bench release-check firmware-check firmware-test firmware-build firmware-upload-blocker-check firmware-clean-check firmware-package firmware-artifact-check firmware-upload-check firmware-device-report firmware-device-check firmware-flash-plan

test:
	go test ./...

preflight:
	go run ./cmd/a21 preflight

doctor:
	go run ./cmd/a21 doctor

gateway:
	go run ./cmd/a21 gateway --addr 127.0.0.1:21080

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

audio-front-end-eval:
	@if [ -n "$(A21_AUDIO_FIXTURE)" ]; then \
		go run ./cmd/a21 audio-front-end-eval --fixture "$(A21_AUDIO_FIXTURE)" --output-dir reports; \
	else \
		go run ./cmd/a21 audio-front-end-eval --mock --output-dir reports; \
	fi

latency-bench:
	go run ./cmd/a21 latency-bench --mock --iterations 5

release-check: verify latency-bench firmware-test firmware-build firmware-upload-blocker-check firmware-package doctor

firmware-check:
	go run ./cmd/a21 firmware-check

firmware-test: firmware-check
	$(PIO) test -d firmware/stackchan -e a21_stackchan_native

firmware-build: firmware-check
	$(PIO) run -d firmware/stackchan

firmware-upload-blocker-check:
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

firmware-clean-check:
	@test -z "$$(git status --porcelain --untracked-files=all)" || (echo "A21 firmware package requires a clean git worktree"; git status --short; exit 2)

firmware-package: firmware-clean-check firmware-build
	go run ./cmd/a21 firmware-package --commit $$(git rev-parse --short=12 HEAD)

firmware-artifact-check:
	@test -n "$(A21_FIRMWARE_ARTIFACT)" || (echo "A21_FIRMWARE_ARTIFACT is required"; exit 2)
	go run ./cmd/a21 firmware-artifact-check --artifact "$(A21_FIRMWARE_ARTIFACT)"

firmware-upload-check:
	@test -n "$(A21_FIRMWARE_ARTIFACT)" || (echo "A21_FIRMWARE_ARTIFACT is required"; exit 2)
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	go run ./cmd/a21 firmware-upload-check --artifact "$(A21_FIRMWARE_ARTIFACT)" --port "$(A21_UPLOAD_PORT)" --commit $$(git rev-parse --short=12 HEAD)

firmware-device-report:
	go run ./cmd/a21 firmware-device-report --gateway-url "$(A21_GATEWAY_URL)" --output-dir reports

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
	go run ./cmd/a21 firmware-flash-plan --artifact "$(A21_FIRMWARE_ARTIFACT)" --port "$(A21_UPLOAD_PORT)" --device-report "$(A21_DEVICE_REPORT)" --device-id "$(A21_DEVICE_ID)" --commit $$(git rev-parse --short=12 HEAD) --max-device-age-ms "$(A21_DEVICE_MAX_AGE_MS)"

verify:
	go test ./...
	git diff --check
