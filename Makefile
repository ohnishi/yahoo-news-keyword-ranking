BINARY := wadai

# go-mecab は #cgo ディレクティブを持たないため、MeCab の位置は
# CGO_LDFLAGS / CGO_CFLAGS で渡す必要がある。mecab-config があれば
# そこから導出する（無い環境でも test-pure だけは動くよう条件付き）。
MECAB_CONFIG ?= mecab-config
ifneq ($(shell command -v $(MECAB_CONFIG) 2>/dev/null),)
CGO_LDFLAGS := $(shell $(MECAB_CONFIG) --libs)
CGO_CFLAGS := -I$(shell $(MECAB_CONFIG) --inc-dir)
export CGO_LDFLAGS
export CGO_CFLAGS
endif

# MeCab (cgo) を必要としないパッケージ。libmecab-dev が無い環境でも
# ここだけはビルド・テストできる。
PURE_PKGS := $(shell go list ./... | grep -v -e '/internal/cli$$' -e '/internal/analyze/mecab$$' -e '^github.com/ohnishi/yahoo-news-keyword-ranking$$')

.PHONY: all build check-mecab test test-pure vet fmt fmt-check tidy clean help

all: fmt-check vet test build

## build: バイナリをビルドする（MeCab のヘッダとライブラリが必要）
build: check-mecab
	go build -o $(BINARY) .

## check-mecab: mecab-config が使えるか確認する
check-mecab:
	@command -v $(MECAB_CONFIG) >/dev/null 2>&1 || { \
		echo "error: $(MECAB_CONFIG) not found."; \
		echo "  macOS:  brew install mecab"; \
		echo "  Debian: sudo apt-get install -y mecab libmecab-dev"; \
		exit 1; }

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
