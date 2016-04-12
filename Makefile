#!/usr/bin/env make

# NAME=cocaine-porto
# VERSION=$(shell git show-ref --head --hash head)

# GO_LDFLAGS=-ldflags "-X `go list ./version`.Version=$(VERSION)"

.DEFAULT: all
.PHONY: fmt vet test

PKGS := $(shell go list ./... | grep -v ^github.com/noxiouz/stout/vendor/ | grep -v ^github.com/noxiouz/stout/version)

all: fmt vet test

vet:
	@echo "+ $@"
	@go vet $(PKGS)

fmt:
	@echo "+ $@"
	@test -z "$$(gofmt -s -l . 2>&1 | grep -v ^vendor/ | tee /dev/stderr)" || \
		(echo >&2 "+ please format Go code with 'gofmt -s'" && false)

test:
	@echo "+ $@"
	@echo $(PKGS)

	@echo "" > coverage.txt

	for pkg in $(PKGS); do go test -coverprofile=profile.out -covermode=atomic $$pkg; \
	if [ -f profile.out ]; then \
		cat profile.out >> coverage.txt; rm  profile.out; \
	fi done; \
