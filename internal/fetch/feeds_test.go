package fetch

import (
	"strings"
	"testing"
)

const feedListHTML = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>RSS</title></head>
<body>
  <a href="/rss/topics/top-picks.xml">主要</a>
  <a href="/rss/media/nhk.xml">NHK</a>
  <a href="/rss/topics/top-picks.xml">主要（重複リンク）</a>
  <a href="/rss/categories/domestic.xml">  国内  </a>
  <a href="/topics/domestic">対象外のリンク</a>
  <a href="https://example.com/other.xml">外部リンク</a>
  <a>href なし</a>
</body></html>`

func TestParseFeedList(t *testing.T) {
	feeds, err := parseFeedList(strings.NewReader(feedListHTML), "text/html; charset=utf-8", FeedListURL)
	if err != nil {
		t.Fatalf("parseFeedList: %v", err)
	}

	// /rss/ 配下のみ、重複は排除、ID昇順。
	want := []struct{ id, name, url string }{
		{"rss/categories/domestic", "国内", "https://news.yahoo.co.jp/rss/categories/domestic.xml"},
		{"rss/media/nhk", "NHK", "https://news.yahoo.co.jp/rss/media/nhk.xml"},
		{"rss/topics/top-picks", "主要", "https://news.yahoo.co.jp/rss/topics/top-picks.xml"},
	}
	if len(feeds) != len(want) {
		t.Fatalf("got %d feeds, want %d: %+v", len(feeds), len(want), feeds)
	}
	for i, w := range want {
		if feeds[i].ID != w.id {
			t.Errorf("feed %d ID = %q, want %q", i, feeds[i].ID, w.id)
		}
		if feeds[i].Name != w.name {
			t.Errorf("feed %d Name = %q, want %q", i, feeds[i].Name, w.name)
		}
		if feeds[i].URL != w.url {
			t.Errorf("feed %d URL = %q, want %q", i, feeds[i].URL, w.url)
		}
	}
}

func TestParseFeedListIsDeterministic(t *testing.T) {
	first, err := parseFeedList(strings.NewReader(feedListHTML), "text/html", FeedListURL)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		got, err := parseFeedList(strings.NewReader(feedListHTML), "text/html", FeedListURL)
		if err != nil {
			t.Fatal(err)
		}
		for j := range got {
			if got[j] != first[j] {
				t.Fatalf("run %d differs at %d: %+v vs %+v", i, j, got[j], first[j])
			}
		}
	}
}

func TestParseFeedListEmptyPage(t *testing.T) {
	feeds, err := parseFeedList(strings.NewReader("<html><body></body></html>"), "text/html", FeedListURL)
	if err != nil {
		t.Fatalf("parseFeedList: %v", err)
	}
	if len(feeds) != 0 {
		t.Errorf("got %d feeds, want 0", len(feeds))
	}
}

func TestFeedID(t *testing.T) {
	tests := []struct {
		name   string
		href   string
		want   string
		wantOK bool
	}{
		{"media feed", "/rss/media/nhk.xml", "rss/media/nhk", true},
		{"no xml suffix", "/rss/media/nhk", "rss/media/nhk", true},
		{"bare rss path", "/rss/", "", false},
		{"empty", "", "", false},
		// フィードIDはWebページ由来の値で、そのまま保存パスに使われる。
		// ディレクトリを遡るパスは受け付けない。
		{"path traversal", "/rss/../../../etc/passwd.xml", "", false},
		{"embedded traversal", "/rss/a/../../b.xml", "", false},
		{"trailing slash", "/rss/media/.xml", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := feedID(tt.href)
			if ok != tt.wantOK {
				t.Fatalf("feedID(%q) ok = %v, want %v (got %q)", tt.href, ok, tt.wantOK, got)
			}
			if ok && got != tt.want {
				t.Errorf("feedID(%q) = %q, want %q", tt.href, got, tt.want)
			}
		})
	}
}
