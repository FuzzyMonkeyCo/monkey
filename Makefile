.PHONY: all update debug lint x test ape

EXE = monkey

GPB ?= 3.6.1
GPB_IMG ?= znly/protoc:0.4.0
GOGO ?= v1.2.1
PROTOC = docker run --rm -v "$$GOPATH:$$GOPATH":ro -v "$$PWD:$$PWD" -w "$$PWD" $(GPB_IMG) -I=. -I=$$GOPATH/pkg/mod/github.com/gogo/protobuf@$(GOGO)/protobuf

all: SHELL = /bin/bash
all: pkg/internal/fm/fuzzymonkey.pb.go lint
	if [[ $$((RANDOM % 10)) -eq 0 ]]; then go vet; fi
	CGO_ENABLED=0 go build -o $(EXE) -ldflags '-s -w' $(if $(wildcard $(EXE)),|| rm $(EXE))

update: SHELL := /bin/bash
update:
	go get -u -a
	go mod tidy
	go mod verify
	[[ 'libprotoc $(GPB)' = "$$(docker run --rm $(GPB_IMG) --version)" ]]
	[[ 2 = $$(git grep gogo/protobuf -- go.sum | wc -l) ]]

latest:
	cat .godownloader.sh | BINDIR=$$HOME/.local/bin sh -ex

devdeps:
	go install -i github.com/wadey/gocovmerge
	go install -i github.com/kyoh86/richgo

pkg/internal/fm/fuzzymonkey.pb.go: pkg/internal/fm/fuzzymonkey.proto
	$(PROTOC) --gogofast_out=plugins=grpc,Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types:. $^
#	FIXME: don't have this github.com/ folder created in the first place
	cat github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm/fuzzymonkey.pb.go >$@

lint:
	go fmt ./...
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
