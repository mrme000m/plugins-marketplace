# bw-plugin Makefile
# Cross-platform build for Bitwarden Multi-Account CLI Manager

BINARY := bw-plugin
VERSION := 1.0.0
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

# Default: build for current platform
.PHONY: all
all: build

.PHONY: build
build:
	go build $(LDFLAGS) -o $(BINARY) .

# Install to ~/bin
.PHONY: install
install: build
	mkdir -p $(HOME)/bin
	cp $(BINARY) $(HOME)/bin/
	ln -sf $(HOME)/bin/$(BINARY) $(HOME)/bin/bwp
	ln -sf $(HOME)/bin/$(BINARY) $(HOME)/bin/bww
	ln -sf $(HOME)/bin/$(BINARY) $(HOME)/bin/bwa
	@echo "Installed to $(HOME)/bin/"
	@echo "  $(HOME)/bin/$(BINARY)"
	@echo "  $(HOME)/bin/bwp (symlink)"
	@echo "  $(HOME)/bin/bww (symlink)"
	@echo "  $(HOME)/bin/bwa (symlink)"

# Uninstall
.PHONY: uninstall
uninstall:
	rm -f $(HOME)/bin/$(BINARY)
	rm -f $(HOME)/bin/bwp
	rm -f $(HOME)/bin/bww
	rm -f $(HOME)/bin/bwa

# Cross-compilation
.PHONY: build-all
build-all: build-darwin build-linux build-windows

.PHONY: build-darwin
build-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY)-darwin-arm64 .

.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY)-linux-arm64 .

.PHONY: build-windows
build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-windows-amd64.exe .

# Clean
.PHONY: clean
clean:
	rm -f $(BINARY) $(BINARY)-*

# Run tests (basic)
.PHONY: test
test:
	go test -v ./...

# Development run
.PHONY: run
run:
	go run . status
