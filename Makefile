.PHONY: all update debug lint x test

EXE = monkey
OS ?= linux darwin windows
ARCH ?= amd64
SHA = sha256.txt
FMT = $(EXE)-{{.OSUname}}-{{.ArchUname}}
DST ?= .

DEP ?= dep-linux-amd64
GODEP = v0.4.1

all: lint vendor
	go generate
	go build -o $(EXE)

x: vendor
	$(if $(wildcard $(EXE)-*-*.$(SHA)),rm $(EXE)-*-*.$(SHA))
	go generate
	gox -os '$(OS)' -arch '$(ARCH)' -output '$(DST)/$(FMT)' -ldflags '-s -w' -verbose .
	cd $(DST) && for bin in $(EXE)-*; do sha256sum $$bin | tee $$bin.$(SHA); done
	$(if $(filter-out .,$(DST)),,sha256sum --check --strict *$(SHA))

update: SHELL := /bin/bash
update:
	[[ $(GODEP) = "$$(basename $$(curl -#fSLo /dev/null -w '%{url_effective}' https://github.com/golang/dep/releases/latest))" ]]
	go generate
	dep ensure -v -update

latest:
	sh -eux <misc/latest.sh

vendor:
	go generate
	dep ensure -v
#	Note: workaround to https://github.com/golang/dep/issues/1554
#	Writes to $GOPATH/bin so keep that in mind...
	for pkg in $$(grep -Eo '"[^"]+",' Gopkg.toml | tr -d '",'); do \
	  cd vendor/$$pkg && go install . && cd - ; \
	done

deps:
	mkdir -p release
	curl -#fSL https://github.com/golang/dep/releases/download/$(GODEP)/$(DEP) -o release/$(DEP)
	curl -#fSL https://github.com/golang/dep/releases/download/$(GODEP)/$(DEP).sha256 -o release/$(DEP).sha256
	sha256sum --check --strict release/$(DEP).sha256
	chmod +x release/$(DEP)
	mv -v release/$(DEP) $$GOPATH/bin/dep
	rm -r release

lint:
	golint -set_exit_status
	./misc/goolint.sh

debug: all
	./$(EXE) validate
	./$(EXE) -vvv fuzz

distclean: clean
	$(if $(wildcard vendor/),rm -r vendor/)
	$(if $(wildcard $(EXE)-*-*.$(SHA)),rm $(EXE)-*-*.$(SHA))
	$(if $(wildcard $(EXE)-*-*),rm $(EXE)-*-*)
clean:
	$(if $(wildcard meta.go),rm meta.go)
	$(if $(wildcard schemas.go),rm schemas.go)
	$(if $(wildcard $(EXE)),rm $(EXE))
	$(if $(wildcard $(EXE).test),rm $(EXE).test)
	$(if $(wildcard *.cov),rm *.cov)
	$(if $(wildcard cov.out),rm cov.out)

test: $(EXE).test
	./ape.sh --version
	gocovmerge *.cov >cov.out
	go tool cover -func cov.out
	rm 0.cov cov.out

# Thanks https://blog.cloudflare.com/go-coverage-with-external-tests
$(EXE).test: lint vendor
	$(if $(wildcard *.cov),rm *.cov)
	go generate
	go test -covermode=count -c

test-cleanup:
	gocovmerge *.cov >cov.out
	go tool cover -func cov.out
	go tool cover -html cov.out
	$(if $(wildcard *.cov),rm *.cov)
