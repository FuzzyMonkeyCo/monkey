.PHONY: debug lint

EXE = monkey

all: lint vendor/
	go generate
	go build -o $(EXE)

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
