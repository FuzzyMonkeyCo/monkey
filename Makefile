.PHONY: debug

EXE = testman

all: $(filter-out schemas.go,$(wildcard *.go)) $(wildcard misc/*.json)
	go generate
	go get .
	go build -o $(EXE)

debug: all
	DEBUG=1 ./$(EXE) test --slow
