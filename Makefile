# Simple shortcuts so we don't have to type the full go commands every time.
# Usage: make <target>   (e.g. "make test")

# on Windows the binary needs the .exe ending, everywhere else it doesn't
ifeq ($(OS),Windows_NT)
BINARY := scope.exe
else
BINARY := scope
endif

.PHONY: build run test race vet fmt fmt-check check clean install

build:
	go build -o $(BINARY) .

run:
	go run . $(ARGS)

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@out=$$(gofmt -l .); \
	if [ -n "$$out" ]; then echo "these files need formatting:"; echo "$$out"; exit 1; fi

# run everything CI runs, before pushing
check: vet fmt-check test

clean:
	go clean
	rm -f $(BINARY)

install:
	go install .
