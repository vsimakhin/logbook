BINARY_NAME="logbook"
XC_OS ?= linux darwin windows
XC_ARCH ?= amd64 arm64
XC_OS ?= linux
XC_ARCH ?= amd64
BIN="./bin"
VERSION ?= $(shell git describe --tags --abbrev=0)
BUILD_TIME=$(shell date -u "+%FT%H:%M:%SZ")

.PHONY: all build test clean

default: all

all: test build

test:
	go test -v ./...

build:
# Windows arm64 doesn't compile correctly with Go <=1.17.2, skip it
	@for OS in $(XC_OS); do \
		for ARCH in $(XC_ARCH); do \
			[ $$OS = "windows" ] && [ $$ARCH = "arm64" ] && continue; \
			mkdir -p $(BIN)/$(BINARY_NAME)-$(VERSION)-$$OS-$$ARCH ; \
			echo Building $$OS/$$ARCH to $(BIN)/$(BINARY_NAME)-$(VERSION)-$$OS-$$ARCH/logbook ; \
			CGO_ENABLED=0 \
			GOOS=$$OS \
			GOARCH=$$ARCH \
			go build \
			-ldflags="-s -w -X 'github.com/vsimakhin/$(BINARY_NAME)/cmd.version=$(VERSION)' -X 'github.com/vsimakhin/$(BINARY_NAME)/cmd.buildTime=$(BUILD_TIME)'" \
			-o=$(BIN)/$(BINARY_NAME)-$(VERSION)-$$OS-$$ARCH/logbook ; \
			[ $$OS = "windows" ] && (cd $(BIN); zip -r $(BINARY_NAME)-$(VERSION)-$$OS-$$ARCH.zip $(BINARY_NAME)-$(VERSION)-$$OS-$$ARCH; cd ../) \
				|| (cd $(BIN); tar czf $(BINARY_NAME)-$(VERSION)-$$OS-$$ARCH.tar.gz $(BINARY_NAME)-$(VERSION)-$$OS-$$ARCH; cd ../) ;\
		done ; \
	done

clean:
	rm -rfv $(BIN)