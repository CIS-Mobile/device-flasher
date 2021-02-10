GOOS_LINUX=linux
GOARCH_LINUX=amd64

GOOS_WINDOWS=windows
GOARCH_WINDOWS=amd64

VERSION=1.2.0

GIT_COMMIT := $(shell git rev-list -1 HEAD)

all : build_linux build_windows
.PHONY : all

build_linux:
	GOOS=$(GOOS_LINUX) GOARCH=$(GOARCH_LINUX) go build -ldflags "-X main.version=$(VERSION) -X 'main.gitCommit=$(GIT_COMMIT)'" .

build_windows:
	GOOS=$(GOOS_WINDOWS) GOARCH=$(GOARCH_WINDOWS) go build -ldflags "-X main.version=$(VERSION) -X 'main.gitCommit=$(GIT_COMMIT)'" .