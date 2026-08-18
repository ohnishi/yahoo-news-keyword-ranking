// Package mecab は MeCab による analyze.Tokenizer 実装を提供する。
//
// cgo 経由で libmecab に依存するのはこのパッケージだけで、
// 解析ロジック本体（internal/analyze）は純粋な Go のままテストできる。
package mecab

import (
	"fmt"
	"os"
	"strings"

	gomecab "github.com/shogo82148/go-mecab"

	"github.com/ohnishi/yahoo-news-keyword-ranking/internal/analyze"
)

// DicDirCandidates は mecab-ipadic-NEologd の一般的なインストール先。
// 先頭から順に探索する。
var DicDirCandidates = []string{
	"/opt/homebrew/lib/mecab/dic/mecab-ipadic-neologd", // Homebrew (Apple Silicon)
	"/usr/local/lib/mecab/dic/mecab-ipadic-neologd",    // Homebrew (Intel) / 手動ビルド
	"/usr/lib/mecab/dic/mecab-ipadic-neologd",          // Debian/Ubuntu
	"/var/lib/mecab/dic/mecab-ipadic-neologd",
}

// DefaultDicDir は候補のうち最初に存在したディレクトリを返す。
// どれも無ければ空文字を返し、その場合は MeCab の既定辞書が使われる。
func DefaultDicDir() string {
	for _, dir := range DicDirCandidates {
		if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
			return dir
		}
	}
	return ""
}

// Tokenizer は MeCab を用いた analyze.Tokenizer の実装。
// 内部の MeCab インスタンスは並行利用できないため、単一ゴルーチンから使うこと。
type Tokenizer struct {
	mecab gomecab.MeCab
}

// 実装漏れをコンパイル時に検出する。
var _ analyze.Tokenizer = (*Tokenizer)(nil)

// New は辞書ディレクトリを指定して Tokenizer を生成する。
// dicDir が空なら MeCab の既定辞書を使う。
func New(dicDir string) (*Tokenizer, error) {
	args := map[string]string{}
	if dicDir != "" {
		if stat, err := os.Stat(dicDir); err != nil || !stat.IsDir() {
			return nil, fmt.Errorf("mecab dictionary directory not found: %s", dicDir)
		}
		args["dicdir"] = dicDir
	}

	m, err := gomecab.New(args)
	if err != nil {
		return nil, fmt.Errorf("initialize mecab (dicdir=%q): %w", dicDir, err)
	}
	return &Tokenizer{mecab: m}, nil
}

// Tokenize は text を形態素へ分割する。
func (t *Tokenizer) Tokenize(text string) ([]analyze.Token, error) {
	node, err := t.mecab.ParseToNode(text)
	if err != nil {
		return nil, fmt.Errorf("mecab parse: %w", err)
	}

	var tokens []analyze.Token
	for ; !node.IsZero(); node = node.Next() {
		tokens = append(tokens, analyze.Token{
			Surface:  node.Surface(),
			Features: strings.Split(node.Feature(), ","),
		})
	}
	return tokens, nil
}

// Close は MeCab インスタンスを解放する。
func (t *Tokenizer) Close() error {
	t.mecab.Destroy()
	return nil
}
