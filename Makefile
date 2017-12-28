.PHONY: debug lint

EXE = testman

all: lint vendor/
	go generate
	go build -o $(EXE)

vendor/:
	go generate
	dep ensure -v

lint:
	golint -set_exit_status

debug: all
	./$(EXE) validate
	./$(EXE) -vvv test
