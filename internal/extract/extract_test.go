package extract

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ohnishi/yahoo-news-keyword-ranking/internal/model"
)

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// rssXML は指定したアイテムを持つ最小のRSS 2.0フィードを組み立てる。
func rssXML(feedTitle string, items ...[3]string) string {
	body := ""
	for _, it := range items {
		body += fmt.Sprintf(
			"<item><title>%s</title><link>%s</link><pubDate>%s</pubDate></item>\n",
			it[0], it[1], it[2])
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>%s</title><link>https://example.com</link>
<description>test feed</description>
%s</channel></rss>`, feedTitle, body)
}

func writeFeedFile(t *testing.T, dir, id, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(id))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCollectKeepsOnlyTargetDate(t *testing.T) {
	dir := t.TempDir()
	writeFeedFile(t, dir, "rss/media/a", rssXML("Feed A",
		[3]string{"当日の記事", "https://example.com/1", "Fri, 18 Dec 2020 10:00:00 +0900"},
		[3]string{"前日の記事", "https://example.com/2", "Thu, 17 Dec 2020 10:00:00 +0900"},
		[3]string{"翌日の記事", "https://example.com/3", "Sat, 19 Dec 2020 10:00:00 +0900"},
	))

	feeds := []model.RSSFeed{{ID: "rss/media/a"}}
	date := time.Date(2020, 12, 18, 0, 0, 0, 0, time.Local)

	got, err := Collect(feeds, dir, date, quietLogger())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d articles, want 1: %+v", len(got), got)
	}
	if got[0].Title != "当日の記事" {
		t.Errorf("title = %q, want 当日の記事", got[0].Title)
	}
	if got[0].Name != "Feed A" {
		t.Errorf("name = %q, want the feed title Feed A", got[0].Name)
	}
}

func TestCollectDeduplicatesByURL(t *testing.T) {
	dir := t.TempDir()
	item := [3]string{"共通の記事", "https://example.com/1", "Fri, 18 Dec 2020 10:00:00 +0900"}
	writeFeedFile(t, dir, "rss/media/a", rssXML("Feed A", item))
	writeFeedFile(t, dir, "rss/media/b", rssXML("Feed B", item))

	feeds := []model.RSSFeed{{ID: "rss/media/a"}, {ID: "rss/media/b"}}
	date := time.Date(2020, 12, 18, 0, 0, 0, 0, time.Local)

	got, err := Collect(feeds, dir, date, quietLogger())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d articles, want 1", len(got))
	}
}

// 以前は map をそのまま走査して書き出していたため、行順が実行ごとに変わっていた。
func TestCollectIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	var items [][3]string
	for i := 0; i < 25; i++ {
		items = append(items, [3]string{
			fmt.Sprintf("記事%02d", i),
			fmt.Sprintf("https://example.com/%02d", i),
			"Fri, 18 Dec 2020 10:00:00 +0900",
		})
	}
	writeFeedFile(t, dir, "rss/media/a", rssXML("Feed A", items...))

	feeds := []model.RSSFeed{{ID: "rss/media/a"}}
	date := time.Date(2020, 12, 18, 0, 0, 0, 0, time.Local)

	first, err := Collect(feeds, dir, date, quietLogger())
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 25 {
		t.Fatalf("got %d articles, want 25", len(first))
	}
	for run := 0; run < 10; run++ {
		got, err := Collect(feeds, dir, date, quietLogger())
		if err != nil {
			t.Fatal(err)
		}
		for i := range got {
			if got[i].URL != first[i].URL {
				t.Fatalf("run %d differs at %d: %s vs %s", run, i, got[i].URL, first[i].URL)
			}
		}
	}
	for i := 1; i < len(first); i++ {
		if first[i-1].URL > first[i].URL {
			t.Fatalf("not sorted by URL: %s before %s", first[i-1].URL, first[i].URL)
		}
	}
}

// RSSリストが更新されて当日のfetchファイルが無いフィードは黙って飛ばす。
func TestCollectSkipsMissingFiles(t *testing.T) {
	dir := t.TempDir()
	writeFeedFile(t, dir, "rss/media/a", rssXML("Feed A",
		[3]string{"記事", "https://example.com/1", "Fri, 18 Dec 2020 10:00:00 +0900"}))

	feeds := []model.RSSFeed{{ID: "rss/media/a"}, {ID: "rss/media/missing"}}
	date := time.Date(2020, 12, 18, 0, 0, 0, 0, time.Local)

	got, err := Collect(feeds, dir, date, quietLogger())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d articles, want 1", len(got))
	}
}

// 壊れたRSSがあっても他のフィードの処理は続行する。
func TestCollectContinuesPastUnparseableFeed(t *testing.T) {
	dir := t.TempDir()
	writeFeedFile(t, dir, "rss/media/broken", "this is not xml at all")
	writeFeedFile(t, dir, "rss/media/ok", rssXML("Feed OK",
		[3]string{"記事", "https://example.com/1", "Fri, 18 Dec 2020 10:00:00 +0900"}))

	feeds := []model.RSSFeed{{ID: "rss/media/broken"}, {ID: "rss/media/ok"}}
	date := time.Date(2020, 12, 18, 0, 0, 0, 0, time.Local)

	got, err := Collect(feeds, dir, date, quietLogger())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d articles, want 1", len(got))
	}
}

// pubDate の無い記事は対象日の記事として扱う。
func TestCollectFallsBackToTargetDateWithoutPubDate(t *testing.T) {
	dir := t.TempDir()
	writeFeedFile(t, dir, "rss/media/a", `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>Feed A</title><link>https://example.com</link>
<description>test</description>
<item><title>日付なし</title><link>https://example.com/1</link></item>
</channel></rss>`)

	feeds := []model.RSSFeed{{ID: "rss/media/a"}}
	date := time.Date(2020, 12, 18, 0, 0, 0, 0, time.Local)

	got, err := Collect(feeds, dir, date, quietLogger())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d articles, want 1", len(got))
	}
}
