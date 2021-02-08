.PHONY: build

GIT_COMMIT := $(shell git rev-list -1 HEAD)

build:
	go build -ldflags "-X main.version=1.2.0 -X 'main.gitCommit=$(GIT_COMMIT)'" .