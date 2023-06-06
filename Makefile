VERSION=1.3.1

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

build_linux:
	GO111MODULE=auto GOOS=linux GOARCH=amd64 go build -o "altOS-flasher_linux-x86_64" -ldflags "-X main.version=$(VERSION) -X 'main.gitCommit=$(GIT_COMMIT)'" .
#TODO: Re-enable this when Google builds adb and fastboot for arm64
#	GO111MODULE=auto GOOS=linux GOARCH=arm64 go build -o "altOS-flasher_linux-arm64" -ldflags "-X main.version=$(VERSION) -X 'main.gitCommit=$(GIT_COMMIT)'" .

build_windows:
	GO111MODULE=auto GOOS=windows GOARCH=amd64 go build -o "altOS-flasher_windows-x86_64.exe" -ldflags "-X main.version=$(VERSION) -X 'main.gitCommit=$(GIT_COMMIT)'" .

build_mac:
	GO111MODULE=auto GOOS=darwin GOARCH=amd64 go build -o "altOS-flasher_darwin-x86_64" -ldflags "-X main.version=$(VERSION) -X 'main.gitCommit=$(GIT_COMMIT)'" .
	GO111MODULE=auto GOOS=darwin GOARCH=arm64 go build -o "altOS-flasher_darwin-arm64" -ldflags "-X main.version=$(VERSION) -X 'main.gitCommit=$(GIT_COMMIT)'" .
	$(LIPO_BIN) -create -output altOS-flasher_darwin-universal altOS-flasher_darwin-x86_64 altOS-flasher_darwin-arm64
	rm -rf altOS-flasher_darwin-x86_64 altOS-flasher_darwin-arm64
