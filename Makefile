.PHONY: all update debug lint x test

OS ?= linux darwin windows
EXE = monkey
ARCH ?= amd64
SHATX = sha256s.txt
FORMAT = $(EXE)-{{.OSUname}}-{{.ArchUname}}

all: lint vendor/
	go generate
	go build -o $(EXE)

x: vendor/
	go generate
	gox -os '$(OS)' -arch '$(ARCH)' -output '$(FORMAT)' -ldflags '-s -w' -verbose .
	for bin in $(EXE)-*; do sha256sum $$bin; done | tee $(SHATX)
	sha256sum --check --strict $(SHATX)

update: SHELL := /bin/bash
update:
	[[ "$$(git grep GODEP= -- .travis.yml | cut -d= -f2)" = "$$(basename $$(curl -sLo /dev/null -w '%{url_effective}' https://github.com/golang/dep/releases/latest) | tr -d v)" ]]
	go generate
	dep ensure -v -update

vendor/:
	go generate
	dep ensure -v

lint:
	golint -set_exit_status
	./misc/goolint.sh

debug: all
	./$(EXE) validate
	./$(EXE) -vvv fuzz

distclean: clean
	$(if $(wildcard vendor/),rm -r vendor/)
clean:
	$(if $(wildcard meta.go),rm meta.go)
	$(if $(wildcard schemas.go),rm schemas.go)
	$(if $(wildcard $(EXE)),rm $(EXE))
	$(if $(wildcard $(EXE).test),rm $(EXE).test)
	$(if $(wildcard $(EXE)-*-*),rm $(EXE)-*-*)
	$(if $(wildcard $(SHATX)),rm $(SHATX))
	$(if $(wildcard *.cov),rm *.cov)
	$(if $(wildcard cov.out),rm cov.out)

test: $(EXE).test
	./ape.sh --version
	gocovmerge *.cov >cov.out
	go tool cover -func cov.out
	rm 0.cov cov.out

# Thanks https://blog.cloudflare.com/go-coverage-with-external-tests
# go get -u github.com/wadey/gocovmerge

test-setup: $(EXE).test
$(EXE).test: lint vendor/
	$(if $(wildcard *.cov),rm *.cov)
	go generate
	go test -covermode=count -c

test-cleanup:
	gocovmerge *.cov >cov.out
	go tool cover -func cov.out
	go tool cover -html cov.out
	$(if $(wildcard *.cov),rm *.cov)
