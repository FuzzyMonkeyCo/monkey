.PHONY: debug

EXE = testman

all:
	golint -set_exit_status
	go generate
	go get .
	go build -o $(EXE)

debug: all
	./$(EXE) validate
	./$(EXE) -vvv test
