.PHONY: debug

EXE = testman

all:
	go generate
	go get .
	go build -o $(EXE)

debug: all
	./$(EXE) validate
	./$(EXE) -vvv test
