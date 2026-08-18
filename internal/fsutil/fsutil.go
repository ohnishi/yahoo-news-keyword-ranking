// Package fsutil はパス解決とファイル生成まわりの小さなヘルパーを提供する。
package fsutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// dirPerm は生成するディレクトリのパーミッション。
const dirPerm = 0o755

// ExpandPath は先頭の "~" をユーザーのホームディレクトリへ展開する。
//
// シェルと違い Go はチルダ展開を行わないため、フラグ経由で受け取った
// パスにはこれを通す必要がある。通さないと "~/Desktop" という名前の
// ディレクトリがカレント直下に作られてしまう。
func ExpandPath(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory for %q: %w", path, err)
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, path[2:]), nil
}

// Create は path のファイルを作成する。既存の場合は切り詰める。
// 親ディレクトリが存在しなければ作成する。
func Create(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return nil, fmt.Errorf("create directory %s: %w", dir, err)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open file %s: %w", path, err)
	}
	return f, nil
}

// WriteFile は path を作成し、write に内容を書かせてから fsync してクローズする。
// write / Sync / Close のいずれかが失敗すればエラーを返す。
//
// 「作成してループで書いて Sync して Close」という定型処理が各段に散らばって
// いたので、ここに一本化している。
func WriteFile(path string, write func(io.Writer) error) (err error) {
	f, err := Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close file %s: %w", path, cerr)
		}
	}()

	if err := write(f); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync file %s: %w", path, err)
	}
	return nil
}
