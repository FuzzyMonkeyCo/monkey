.PHONY: debug

EXE = manlion

all: $(filter-out schemas.go,$(wildcard *.go)) $(wildcard misc/*.json)
	go generate
	go build -o $(EXE)

debug: all
	./$(EXE) test --slow
