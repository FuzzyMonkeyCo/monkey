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

DEP ?= dep-linux-amd64
GODEP = v0.4.1
GPB   = v3.5.1

all: lint vendor
	protoc --go_out=. *.proto
	go generate
	$(if $(wildcard $(EXE)),rm $(EXE))
	go build -o $(EXE)

x: vendor
	$(if $(wildcard $(EXE)-*-*.$(SHA)),rm $(EXE)-*-*.$(SHA))
	go generate
	CGO_ENABLED=0 gox -output '$(DST)/$(FMT)' -ldflags '-s -w' -verbose -osarch "$$(echo $(OSARCH))" .
	cd $(DST) && for bin in $(EXE)-*; do sha256sum $$bin | tee $$bin.$(SHA); done
	$(if $(filter-out .,$(DST)),,sha256sum --check --strict *$(SHA))

update: SHELL := /bin/bash
update:
	[[ $(GODEP) = "$$(basename $$(curl -#fSLo /dev/null -w '%{url_effective}' https://github.com/golang/dep/releases/latest))" ]]
	[[ $(GPB)   = "$$(basename $$(curl -#fSLo /dev/null -w '%{url_effective}' https://github.com/google/protobuf/releases/latest))" ]]
	grep -F GPB
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
	  echo $$pkg && cd vendor/$$pkg && go install . && cd - ; \
	done

deps: dep.GODEP

dep.GODEP:
	mkdir -p release
	curl -#fSL https://github.com/golang/dep/releases/download/$(GODEP)/$(DEP) -o release/$(DEP)
	curl -#fSL https://github.com/golang/dep/releases/download/$(GODEP)/$(DEP).sha256 -o release/$(DEP).sha256
	sha256sum --check --strict release/$(DEP).sha256
	chmod +x release/$(DEP)
	mv -v release/$(DEP) $$GOPATH/bin/dep
	rm -r release

dep.GPB: bin = protoc
dep.GPB: url = https://github.com/google/protobuf/releases
dep.GPB: pre = /usr/local
dep.GPB:
	( cd /tmp \
	  && lat=$(GPB) \
	  && curl -#fSLo protoc.zip $(url)/download/$$lat/$(bin)-$${lat##v}-linux-$$(uname -m).zip \
	  && unzip protoc.zip -d protoc \
	  && sudo mv -v protoc/bin/* $(pre)/bin/ \
	  && sudo mv -v protoc/include/* $(pre)/include/ \
	  && sudo chown -R $$USER:$$USER $(pre)/bin $(pre)/include \
	)

lint:
	gofmt -s -w *.go misc/*.go
	golint -set_exit_status
	./misc/goolint.sh

debug: all
	./$(EXE) lint
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

test: SHELL = /bin/bash -o pipefail
test: all
	go test . | richgo testfilter
test.ci: all
	go test -v -race .

ape: $(EXE).test
	./ape.sh --version
	gocovmerge *.cov >cov.out
	go tool cover -func cov.out
	rm 0.cov cov.out

# Thanks https://blog.cloudflare.com/go-coverage-with-external-tests
$(EXE).test: lint vendor
	$(if $(wildcard *.cov),rm *.cov)
	go generate
	go test -covermode=count -c

ape-cleanup:
	gocovmerge *.cov >cov.out
	go tool cover -func cov.out
	go tool cover -html cov.out
	$(if $(wildcard *.cov),rm *.cov)
