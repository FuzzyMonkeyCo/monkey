.PHONY: all update debug lint test ape

EXE = monkey

GPB ?= 3.6.1
GPB_IMG ?= znly/protoc:0.4.0
GOGO ?= v1.3.1
RUN ?= docker run --rm --user $$(id -u):$$(id -g)
PROTOC = $(RUN) -v "$$GOPATH:$$GOPATH":ro -v "$$PWD:$$PWD" -w "$$PWD" $(GPB_IMG) -I=. -I=$$GOPATH/pkg/mod/github.com/gogo/protobuf@$(GOGO)/protobuf
PROTOLOCK ?= $(RUN) -v "$$PWD":/protolock -w /protolock nilslice/protolock

all: SHELL := /bin/bash
all: pkg/internal/fm/fuzzymonkey.pb.go lint
	CGO_ENABLED=0 go build -o $(EXE) -ldflags '-s -w' $(if $(wildcard $(EXE)),|| (rm $(EXE) && false))
	[[ 1 -ne $$(git status --porcelain -- README.md | grep -Ec '^.[^ ]') ]] && cat <(head -n9 README.md) <(./$(EXE) -h) <(tail -n +51 README.md) >_ && mv _ README.md

update: SHELL := /bin/bash
update:
	go get -u -a -v ./...
	go mod tidy
	go mod verify
	[[ 'libprotoc $(GPB)' = "$$($(RUN) $(GPB_IMG) --version)" ]]
	git grep -F 'github.com/gogo/protobuf $(GOGO)' -- go.mod

latest: bindir ?= $$HOME/.local/bin
latest:
	cat .godownloader.sh | BINDIR=$(bindir) sh -ex
	$(bindir)/$(EXE) --version

devdeps:
	go install -i github.com/wadey/gocovmerge
	go install -i github.com/kyoh86/richgo

pkg/internal/fm/fuzzymonkey.pb.go: pkg/internal/fm/fuzzymonkey.proto
	cd pkg/internal/fm && $(PROTOLOCK) commit
	$(PROTOC) --gogofast_out=plugins=grpc,Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types:. $^
	mv github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm/fuzzymonkey.pb.go $@
	git clean -xdff -- ./github.com

lint: SHELL = /bin/bash
lint:
	go fmt ./...
	./misc/goolint.sh
	go vet ./...

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
