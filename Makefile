.PHONY: all update debug lint x test ape

EXE = monkey

GPB ?= 3.6.1
GPB_IMG ?= znly/protoc:0.4.0

all: lib/messages.pb.go lint
	CGO_ENABLED=0 go build -o $(EXE) -ldflags '-s -w' $(if $(wildcard $(EXE)),|| rm $(EXE))

update: SHELL := /bin/bash
update:
	[[ 'libprotoc $(GPB)' = "$$(docker run --rm $(GPB_IMG) --version)" ]]
	go get -u -a
	go mod tidy
	go mod verify

latest:
	sh -eux <misc/latest.sh

devdeps:
	go install -i github.com/wadey/gocovmerge
	go install -i github.com/kyoh86/richgo

lib/messages.pb.go: PROTOC ?= docker run --rm -v "$$PWD:$$PWD" -w "$$PWD" $(GPB_IMG) -I=.
lib/messages.pb.go: lib/messages.proto
	$(PROTOC) --gogofast_out=. $^
#	FIXME: don't have this github.com/ folder created in the first place
	cat github.com/FuzzyMonkeyCo/monkey/lib/messages.pb.go >$@

lint:
	gofmt -s -w *.go */*.go
	./misc/goolint.sh

debug: all
	./$(EXE) lint
	./$(EXE) -vvv fuzz

distclean: clean
	$(if $(wildcard dist/),rm -r dist/)
clean:
	$(if $(wildcard $(EXE)),rm $(EXE))
	$(if $(wildcard $(EXE).test),rm $(EXE).test)
	$(if $(wildcard *.cov),rm *.cov)
	$(if $(wildcard cov.out),rm cov.out)

test: SHELL = /bin/bash -o pipefail
test: all
	richgo test -count 10 ./...
test.ci: all
	go test -v -race ./...

ape: $(EXE).test
	./ape.sh --version
	gocovmerge *.cov >cov.out
	go tool cover -func cov.out
	rm 0.cov cov.out

# Thanks https://blog.cloudflare.com/go-coverage-with-external-tests
$(EXE).test: lint
	$(if $(wildcard *.cov),rm *.cov)
	go test -covermode=count -c

ape-cleanup:
	gocovmerge *.cov >cov.out
	go tool cover -func cov.out
	go tool cover -html cov.out
	$(if $(wildcard *.cov),rm *.cov)
