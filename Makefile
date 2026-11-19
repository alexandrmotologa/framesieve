.PHONY: all build test bench clean web-build run-demo

BINARY_NAME=framesieve
ifeq ($(OS),Windows_NT)
	BINARY_EXT=.exe
else
	BINARY_EXT=
endif

all: web-build build

web-build:
	cd web && npm install && npm run build

build:
	go build -ldflags="-s -w" -o $(BINARY_NAME)$(BINARY_EXT) ./cmd/framesieve

test:
	go test -v ./internal/...

bench:
	./$(BINARY_NAME)$(BINARY_EXT) bench

run-demo: build
	./$(BINARY_NAME)$(BINARY_EXT) demo

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	rm -rf web/dist
	rm -f framesieve.db*
