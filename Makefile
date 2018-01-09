.PHONY: debug lint

EXE = monkey

all: lint vendor/
	go generate
	go build -o $(EXE)

update:
	dep ensure -v -update

vendor/:
	go generate
	dep ensure -v

lint:
	golint -set_exit_status

debug: all
	./$(EXE) validate
	./$(EXE) -vvv fuzz

clean:
	$(if $(wildcard vendor/),rm -r vendor/)
	$(if $(wildcard schemas.go),rm schemas.go)
	$(if $(wildcard $(EXE)),rm $(EXE))
