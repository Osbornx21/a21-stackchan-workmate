PLATFORMIO_CORE_DIR := $(CURDIR)/.a21-tools/platformio-core
PIO := env PLATFORMIO_CORE_DIR="$(PLATFORMIO_CORE_DIR)" .a21-tools/platformio-venv/bin/pio

.PHONY: test verify preflight doctor gateway firmware-check firmware-build firmware-package

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

firmware-build: firmware-check
	$(PIO) run -d firmware/stackchan

firmware-package: firmware-build
	go run ./cmd/a21 firmware-package --commit $$(git rev-parse --short=12 HEAD)

verify:
	go test ./...
	git diff --check
