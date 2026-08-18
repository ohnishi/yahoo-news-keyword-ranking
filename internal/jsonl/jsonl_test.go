package jsonl

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ohnishi/yahoo-news-analysis/internal/model"
)

func TestWriteReadRoundTrip(t *testing.T) {
	want := []model.RSSFeed{
		{ID: "rss/media/nhk", Name: "NHK", URL: "https://news.yahoo.co.jp/rss/media/nhk.xml"},
		{ID: "rss/topics/top-picks", Name: "主要", URL: "https://news.yahoo.co.jp/rss/topics/top-picks.xml"},
	}

	path := filepath.Join(t.TempDir(), "nested", "rss.jsonl")
	if err := Write(path, want); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := Read[model.RSSFeed](path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d feeds, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("feed %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestEncodeProducesOneObjectPerLine(t *testing.T) {
	var sb strings.Builder
	feeds := []model.RSSFeed{{ID: "a"}, {ID: "b"}}
	if err := Encode(&sb, feeds); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(sb.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), sb.String())
	}
	if !strings.HasSuffix(sb.String(), "\n") {
		t.Error("output should end with a newline")
	}
}

func TestDecodeEmptyInput(t *testing.T) {
	got, err := Decode[model.RSSFeed](strings.NewReader(""))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d elements, want 0", len(got))
	}
}

func TestDecodeReportsElementIndexOnMalformedInput(t *testing.T) {
	in := `{"id":"a"}` + "\n" + `{"id":` + "\n"
	_, err := Decode[model.RSSFeed](strings.NewReader(in))
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "element 1") {
		t.Errorf("error = %v, want it to identify element 1", err)
	}
}

func TestReadMissingFile(t *testing.T) {
	_, err := Read[model.RSSFeed](filepath.Join(t.TempDir(), "missing.jsonl"))
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}
