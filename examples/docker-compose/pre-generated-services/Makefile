.PHONY: generate tidy test lint check-fmt

generate:
	go generate ./...

tidy:
	go mod tidy

test:
	go test -cover ./...

lint:
	go fmt ./...
	go vet ./...

check-fmt:
	@UNFORMATTED=$$(gofmt -l .); \
	if [ -n "$$UNFORMATTED" ]; then \
		echo >&2 "ERROR: The following files are not formatted:"; \
		echo "$$UNFORMATTED"; \
		exit 1; \
	fi

