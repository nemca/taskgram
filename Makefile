PROJECTNAME := taskgram

.PHONY: build clean help

all: build

## build: Compile the binary.
build:
	go build -o $(PROJECTNAME) cmd/main.go

## clean: Cleanup binary.
clean:
	-@rm -f $(PROJECTNAME)

## help: Show this message.
help: Makefile
	@echo "Available targets:"
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
