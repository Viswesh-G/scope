# Simple shortcuts so we don't have to type the full go commands every time.
# Usage: make <target>   (e.g. "make test")

# on Windows the binary needs the .exe ending, everywhere else it doesn't
ifeq ($(OS),Windows_NT)
BINARY := scp.exe
else
BINARY := scp
endif

.PHONY: build run test race vet fmt fmt-check check bench cover vuln complete clean install

build:
	go build -o $(BINARY) .

run:
	go run . $(ARGS)

test:
	go test ./...

race:
	go test -race ./...

bench:
	go test -bench=. -benchmem ./internal/search/

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

vuln:
	@command -v govulncheck >/dev/null 2>&1 || (echo "govulncheck not found. Install it with: go install golang.org/x/vuln/cmd/govulncheck@latest" && exit 1)
	govulncheck ./...

complete:
	# Generate shell completion scripts for scp (e.g., bash, zsh, fish, powershell).
	# Usage: make complete
	./$(BINARY) completion > completion.sh

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@out=$$(gofmt -l .); \
	if [ -n "$$out" ]; then echo "these files need formatting:"; echo "$$out"; exit 1; fi

# run everything CI runs, before pushing
check: vet fmt-check vuln test

clean:
	go clean
	rm -f $(BINARY)

install:
	go install .
