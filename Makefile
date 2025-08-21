.PHONY: build clean
BIN=ghostredact
build:
	mkdir -p dist
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o dist/$(BIN) ./cmd/ghostredact
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o dist/$(BIN).exe ./cmd/ghostredact
clean:
	rm -rf dist
