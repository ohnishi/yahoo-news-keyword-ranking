package fsutil

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("home directory unavailable: %v", err)
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"tilde only", "~", home},
		{"tilde with subdir", "~/Desktop/fetch", filepath.Join(home, "Desktop/fetch")},
		{"absolute path untouched", "/tmp/foo", "/tmp/foo"},
		{"relative path untouched", "foo/bar", "foo/bar"},
		{"tilde not at start", "foo/~/bar", "foo/~/bar"},
		{"tilde-prefixed name is not a home reference", "~user/foo", "~user/foo"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandPath(tt.in)
			if err != nil {
				t.Fatalf("ExpandPath(%q) returned error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestCreateMakesParentDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "c.txt")
	f, err := Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func TestCreateTruncatesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")
	if err := os.WriteFile(path, []byte("stale content"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteFile(path, func(w io.Writer) error {
		_, err := io.WriteString(w, "new")
		return err
	}); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "new" {
		t.Errorf("content = %q, want %q", b, "new")
	}
}

func TestWriteFilePropagatesWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")
	sentinel := errors.New("boom")

	err := WriteFile(path, func(io.Writer) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("WriteFile error = %v, want %v", err, sentinel)
	}
}

func TestWriteFileFailsOnUnwritableParent(t *testing.T) {
	// 既存の通常ファイルを親ディレクトリとして使わせ、MkdirAll を失敗させる。
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	err := WriteFile(filepath.Join(blocker, "child.txt"), func(io.Writer) error { return nil })
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "create directory") {
		t.Errorf("error = %v, want it to mention directory creation", err)
	}
}
