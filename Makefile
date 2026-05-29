.PHONY: test verify preflight doctor gateway

test:
	go test ./...

preflight:
	go run ./cmd/a21 preflight

doctor:
	go run ./cmd/a21 doctor

gateway:
	go run ./cmd/a21 gateway --addr 127.0.0.1:21080

verify:
	go test ./...
	git diff --check
