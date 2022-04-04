PROJECTNAME := taskgram
GOPATH := $(shell go env GOPATH)

.PHONY: build install clean help

all: build

## build: Compile the binary.
build:
	go build -o $(PROJECTNAME) cmd/main.go

## install: Install to $GOBIN path.
install: build
	install $(PROJECTNAME) $(GOPATH)/bin

## clean: Cleanup binary.
clean:
	-@rm -f $(PROJECTNAME)

## help: Show this message.
help: Makefile
	@echo "Available targets:"
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
