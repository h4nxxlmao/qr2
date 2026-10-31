.PHONY: test vet fmt wasm wasm-tinygo corpus verify-corpus

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	@files="$$(gofmt -l .)"; if [ -n "$$files" ]; then echo "gofmt required:"; echo "$$files"; exit 1; fi

wasm:
	GOOS=js GOARCH=wasm go build -o qr2.wasm ./cmd/engine

wasm-tinygo:
	tinygo build -target=wasm -o qr2.wasm ./cmd/engine

corpus:
	go run ./cmd/corpus -out testdata/corpus

verify-corpus: corpus
	python tools/decode_corpus.py testdata/corpus
