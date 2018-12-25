.PHONY: all update debug lint x test ape

EXE = monkey
OSARCH ?= \
  windows/386 windows/amd64 \
   darwin/386  darwin/amd64 \
    linux/386   linux/amd64   linux/arm linux/arm64 linux/mips linux/mipsle \
  freebsd/386 freebsd/amd64 freebsd/arm \
   netbsd/386  netbsd/amd64  netbsd/arm \
  openbsd/386 openbsd/amd64 \
    plan9/386   plan9/amd64 \
              solaris/amd64
SHA = sha256.txt
FMT = $(EXE)-{{.OSUname}}-{{.ArchUname}}
LNX = $(EXE)-Linux-x86_64
DST ?= .

GPB ?= 3.5.1
GPB_IMG ?= znly/protoc:0.3.0

all: lint gpb
	go generate
	$(if $(wildcard $(EXE)),rm $(EXE))
	go build -o $(EXE)

x:
	$(if $(wildcard $(EXE)-*-*.$(SHA)),rm $(EXE)-*-*.$(SHA))
	go generate
	CGO_ENABLED=0 gox -output '$(DST)/$(FMT)' -ldflags '-s -w' -verbose -osarch "$$(echo $(OSARCH))" .
	cd $(DST) && for bin in $(EXE)-*; do sha256sum $$bin | tee $$bin.$(SHA); done
	$(if $(filter-out .,$(DST)),,sha256sum --check --strict *$(SHA))

update: SHELL := /bin/bash
update:
	[[ 'libprotoc $(GPB)' = "$$(docker run --rm $(GPB_IMG) --version)" ]]
	go generate

latest:
	sh -eux <misc/latest.sh

deps:
#	Writes to $GOPATH/bin so keep that in mind...
	go install -i github.com/fenollp/gox
	go install -i golang.org/x/lint/golint
	go install -i honnef.co/go/tools/cmd/megacheck
	go install -i github.com/wadey/gocovmerge
	go install -i github.com/kyoh86/richgo
	go install -i github.com/golang/protobuf/protoc-gen-go

gpb: lib/messages.proto
	docker run --rm -v $$PWD:$$PWD -w $$PWD $(GPB_IMG) --go_out=. -I. $^

lint:
	gofmt -s -w *.go lib/*.go
	golint -set_exit_status
	./misc/goolint.sh

debug: all
	./$(EXE) lint
	./$(EXE) -vvv fuzz

distclean: clean
	$(if $(wildcard $(EXE)-*-*.$(SHA)),rm $(EXE)-*-*.$(SHA))
	$(if $(wildcard $(EXE)-*-*),rm $(EXE)-*-*)
clean:
	$(if $(wildcard meta.go),rm meta.go)
	$(if $(wildcard $(EXE)),rm $(EXE))
	$(if $(wildcard $(EXE).test),rm $(EXE).test)
	$(if $(wildcard *.cov),rm *.cov)
	$(if $(wildcard cov.out),rm cov.out)

test: SHELL = /bin/bash -o pipefail
test: all
	go test ./... | richgo testfilter
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
	go generate
	go test -covermode=count -c

ape-cleanup:
	gocovmerge *.cov >cov.out
	go tool cover -func cov.out
	go tool cover -html cov.out
	$(if $(wildcard *.cov),rm *.cov)
