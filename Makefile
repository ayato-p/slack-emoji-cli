.PHONY: build install wasm clean

build:
	go build -o emo .

install:
	go install .

# WASM ビルド: dist/ に emoji.wasm, wasm_exec.js, index.html を出力する
wasm:
	mkdir -p dist
	GOOS=js GOARCH=wasm go build -o dist/emoji.wasm ./wasm/
	cp "$$(go env GOROOT)/misc/wasm/wasm_exec.js" dist/
	cp web/index.html dist/

clean:
	rm -f emo
	rm -rf dist/
