.PHONY: test verify preflight doctor

test:
	go test ./...

preflight:
	go run ./cmd/a21 preflight

doctor:
	go run ./cmd/a21 doctor

verify:
	go test ./...
	git diff --check
