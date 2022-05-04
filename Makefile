VERSION=1.2.1

GIT_COMMIT := $(shell git rev-list -1 HEAD)

all : build_linux build_windows build_mac
.PHONY : all

OS := $(shell uname)
ifeq ($(OS),Darwin)
    LIPO_BIN := lipo
else
    ifeq ($(OS),Linux)
        LIPO_BIN := ./tools/lipo
    endif
endif

# TEMP: For some reason Darwin arm64 compile is failing on Linux. Perhaps needs the right toolchain
ARCH := $(shell uname -p)
ifeq ($(OS),Darwin)
    compile_darwin_arm64 := true
else
    compile_darwin_arm64 := false
endif

build_linux:
	GOOS=linux GOARCH=amd64 go build -o "altOS-flasher_linux-x86_64" -ldflags "-X main.version=$(VERSION) -X 'main.gitCommit=$(GIT_COMMIT)'" .
	GOOS=linux GOARCH=arm64 go build -o "altOS-flasher_linux-arm64" -ldflags "-X main.version=$(VERSION) -X 'main.gitCommit=$(GIT_COMMIT)'" .

build_windows:
	GOOS=windows GOARCH=amd64 go build -o "altOS-flasher_windows-x86_64.exe" -ldflags "-X main.version=$(VERSION) -X 'main.gitCommit=$(GIT_COMMIT)'" .

build_mac:
	GOOS=darwin GOARCH=amd64 go build -o "altOS-flasher_darwin-x86_64" -ldflags "-X main.version=$(VERSION) -X 'main.gitCommit=$(GIT_COMMIT)'" .
ifeq ($(compile_darwin_arm64),true)
	GOOS=darwin GOARCH=arm64 go build -o "altOS-flasher_darwin-arm64" -ldflags "-X main.version=$(VERSION) -X 'main.gitCommit=$(GIT_COMMIT)'" .
	$(LIPO_BIN) -create -output altOS-flasher_darwin-universal altOS-flasher_darwin-x86_64 altOS-flasher_darwin-arm64
	rm -rf altOS-flasher_darwin-x86_64 altOS-flasher_darwin-arm64
endif
