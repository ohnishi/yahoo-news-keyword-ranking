BINARY := yahoo-news-analysis

# MeCab (cgo) を必要としないパッケージ。libmecab-dev が無い環境でも
# ここだけはビルド・テストできる。
PURE_PKGS := $(shell go list ./... | grep -v -e '/internal/cli$$' -e '/internal/analyze/mecab$$' -e '^github.com/ohnishi/yahoo-news-analysis$$')

.PHONY: all build test test-pure vet fmt fmt-check tidy clean help

all: fmt-check vet test build

## build: バイナリをビルドする（MeCab のヘッダが必要）
build:
	go build -o $(BINARY) .

## test: 全パッケージのテストを実行する（MeCab のヘッダが必要）
test:
	go test -race ./...

## test-pure: MeCab を必要としないパッケージだけテストする
test-pure:
	go test -race $(PURE_PKGS)

## vet: go vet を実行する
vet:
	go vet ./...

## fmt: gofmt を適用する
fmt:
	gofmt -s -w .

## fmt-check: gofmt 差分があれば失敗する
fmt-check:
	@diff=$$(gofmt -s -l .); \
	if [ -n "$$diff" ]; then echo "gofmt needed:"; echo "$$diff"; exit 1; fi

## tidy: go.mod / go.sum を整理する
tidy:
	go mod tidy

## clean: ビルド成果物を削除する
clean:
	rm -f $(BINARY)

## help: ターゲット一覧を表示する
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //'
