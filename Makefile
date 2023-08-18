.PHONY: all update debug lint test ape

EXE ?= monkey

all: pkg/internal/fm/fuzzymonkey.pb.go make_README.sh README.md lint
	CGO_ENABLED=0 go build -o $(EXE) $(if $(wildcard $(EXE)),|| (rm $(EXE) && false))
	cat .gitignore >.dockerignore && echo /.git >>.dockerignore
	./$(EXE) fmt -w && ./make_README.sh

update: SHELL := /bin/bash
update:
	go get -u -a -v ./...
	go mod tidy
	go mod verify

latest: bindir ?= $$HOME/.local/bin
latest:
	cat .godownloader.sh | BINDIR=$(bindir) sh -ex
	$(bindir)/$(EXE) --version

devdeps:
	go install -i github.com/wadey/gocovmerge
	go install -i github.com/kyoh86/richgo

pkg/internal/fm/fuzzymonkey.pb.go: pkg/internal/fm/fuzzymonkey.proto
	docker buildx bake ci-check--protolock ci-check--protoc #ci-check--protolock-force
	touch $@

lint:
	go fmt ./...
	./golint.sh
	! git grep -F log. pkg/cwid/
	go vet ./...

debug: all
	./$(EXE) lint
	./$(EXE) fuzz --exclude-tags=failing #--progress=bar

distclean: clean
	$(if $(wildcard dist/),rm -r dist/)
clean:
	$(if $(wildcard $(EXE)),rm $(EXE))
	$(if $(wildcard $(EXE).test),rm $(EXE).test)
	$(if $(wildcard *.cov),rm *.cov)
	$(if $(wildcard cov.out),rm cov.out)

test: SHELL = /bin/bash -o pipefail
test: all
	echo 42 | ./$(EXE) schema --validate-against=#/components/schemas/PostId
	! ./$(EXE) exec repl <<<'assert that("malformed" != 42)'
	./$(EXE) exec repl <<<'{"Hullo":41,"how\"":["do","0".isdigit(),{},[],set([13.37])],"you":"do"}'
	./$(EXE) exec repl <<<'assert that("this").is_not_equal_to("that")'
	./$(EXE) exec repl <<<'x = 1.0; print(str(x)); print(str(int(x)))'
	! ./$(EXE) exec repl <<<'assert that(42).is_not_equal_to(42)'
	[[ 1 = "$$(./$(EXE) exec start 2>&1 | wc -l)" ]]
	[[ 6 = "$$(./$(EXE) exec reset 2>&1 | wc -l)" ]]
	[[ 1 = "$$(./$(EXE) exec stop 2>&1 | wc -l)" ]]
	richgo test -race -covermode=atomic ./...

ci:
	docker buildx bake ci-checks

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
