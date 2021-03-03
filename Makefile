PKG := github.com/opendexnetwork/opendex-launcher

GO_BIN := ${GOPATH}/bin

GOBUILD := go build -v

VERSION := latest
COMMIT := $(shell git rev-parse HEAD)
ifeq ($(OS),Windows_NT)
	TIMESTAMP := $(shell powershell.exe scripts\get_timestamp.ps1)
else
	TIMESTAMP := $(shell date +%s)
endif

ifeq ($(GOOS), windows)
	OUTPUT := opendex-launcher.exe
else
	OUTPUT := opendex-launcher
endif


LDFLAGS := -ldflags "-w -s \
-X $(PKG)/build.Version=$(VERSION) \
-X $(PKG)/build.GitCommit=$(COMMIT) \
-X $(PKG)/build.Timestamp=$(TIMESTAMP)"

default: build

build:
	$(GOBUILD) $(LDFLAGS)

zip:
	zip --junk-paths opendex-launcher.zip $(OUTPUT)

clean:
	rm -f opendex-launcher
	rm -f opendex-launcher.zip

.PHONY: build
