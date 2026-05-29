PLATFORMIO_CORE_DIR := $(CURDIR)/.a21-tools/platformio-core
PIO := env PLATFORMIO_CORE_DIR="$(PLATFORMIO_CORE_DIR)" .a21-tools/platformio-venv/bin/pio

.PHONY: test verify preflight doctor gateway firmware-check firmware-test firmware-build firmware-package firmware-artifact-check firmware-upload-check

test:
	go test ./...

preflight:
	go run ./cmd/a21 preflight

doctor:
	go run ./cmd/a21 doctor

gateway:
	go run ./cmd/a21 gateway --addr 127.0.0.1:21080

firmware-check:
	go run ./cmd/a21 firmware-check

firmware-test: firmware-check
	$(PIO) test -d firmware/stackchan -e a21_stackchan_native

firmware-build: firmware-check
	$(PIO) run -d firmware/stackchan

firmware-package: firmware-build
	go run ./cmd/a21 firmware-package --commit $$(git rev-parse --short=12 HEAD)

firmware-artifact-check:
	@test -n "$(A21_FIRMWARE_ARTIFACT)" || (echo "A21_FIRMWARE_ARTIFACT is required"; exit 2)
	go run ./cmd/a21 firmware-artifact-check --artifact "$(A21_FIRMWARE_ARTIFACT)"

firmware-upload-check:
	@test -n "$(A21_FIRMWARE_ARTIFACT)" || (echo "A21_FIRMWARE_ARTIFACT is required"; exit 2)
	@test -n "$(A21_UPLOAD_PORT)" || (echo "A21_UPLOAD_PORT is required"; exit 2)
	go run ./cmd/a21 firmware-upload-check --artifact "$(A21_FIRMWARE_ARTIFACT)" --port "$(A21_UPLOAD_PORT)" --commit $$(git rev-parse --short=12 HEAD)

verify:
	go test ./...
	git diff --check
