// Package jsonl は JSON Lines（1行1JSONオブジェクト）形式の読み書きを提供する。
//
// 以前は「ファイルを開いて json.Decoder で d.More() ループ」という同一の
// コードが型ごとに複製されていた。ジェネリクスで1箇所にまとめている。
package jsonl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ohnishi/yahoo-news-analysis/internal/fsutil"
)

// Read は path の JSON Lines ファイルを読み、各要素を T にデコードして返す。
func Read[T any](path string) ([]T, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	vs, err := Decode[T](f)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return vs, nil
}

// Decode は r から JSON Lines を読み、各要素を T にデコードして返す。
func Decode[T any](r io.Reader) ([]T, error) {
	var vs []T
	d := json.NewDecoder(r)
	for d.More() {
		var v T
		if err := d.Decode(&v); err != nil {
			return nil, fmt.Errorf("decode element %d: %w", len(vs), err)
		}
		vs = append(vs, v)
	}
	return vs, nil
}

// Write は vs を JSON Lines として path へ書き出す。
// 親ディレクトリが無ければ作成する。
func Write[T any](path string, vs []T) error {
	return fsutil.WriteFile(path, func(w io.Writer) error {
		return Encode(w, vs)
	})
}

// Encode は vs を JSON Lines として w へ書き出す。
func Encode[T any](w io.Writer, vs []T) error {
	bw := bufio.NewWriter(w)
	enc := json.NewEncoder(bw)
	for i, v := range vs {
		// json.Encoder は要素ごとに改行を付けるので JSON Lines になる。
		if err := enc.Encode(v); err != nil {
			return fmt.Errorf("encode element %d: %w", i, err)
		}
	}
	return bw.Flush()
}
