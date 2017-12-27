#!/usr/bin/env make

NAME=cocaine-isolate-daemon
BUILDDT=$(shell date -u +%F@%H:%M:%S)
VERSION=$(shell git show-ref --head --hash head)
TAG=$(shell git describe --tags --always)
DEBVER=$(shell dpkg-parsechangelog | sed -n -e 's/^Version: //p')
LDFLAGS=-ldflags "-X github.com/noxiouz/stout/version.GitTag=${TAG} -X github.com/noxiouz/stout/version.Version=${DEBVER} -X github.com/noxiouz/stout/version.Build=${BUILDDT} -X github.com/noxiouz/stout/version.GitHash=${VERSION}"


.DEFAULT: all
.PHONY: fmt vet test gen_msgp

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
	@echo "" > coverage.txt
	@set -e; for pkg in $(PKGS); do go test -coverprofile=profile.out -covermode=atomic $$pkg; \
	if [ -f profile.out ]; then \
		cat profile.out >> coverage.txt; rm  profile.out; \
	fi done; \

build: gen_msgp
	@echo "+ $@"
	go build ${LDFLAGS} -o ${NAME} github.com/noxiouz/stout/cmd/stout

build_travis_release: gen_msgp
	@echo "+ $@"
	env GOOS="linux" go build ${LDFLAGS} -o ${NAME} github.com/noxiouz/stout/cmd/stout
	env GOOS="darwin" go build ${LDFLAGS} -o ${NAME}_osx github.com/noxiouz/stout/cmd/stout

gen_msgp:
	@cd ./isolate; go generate; cd ..
