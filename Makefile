EXE = ./manlion

all:
	go generate
	go build

debug: all
	$(EXE) test --slow .
