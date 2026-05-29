.PHONY: test verify preflight

test:
	go test ./...

preflight:
	go run ./cmd/a21 preflight

verify:
	go test ./...
	git diff --check
