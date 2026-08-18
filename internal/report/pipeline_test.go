package report_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ohnishi/yahoo-news-analysis/internal/analyze"
	"github.com/ohnishi/yahoo-news-analysis/internal/daterange"
	"github.com/ohnishi/yahoo-news-analysis/internal/extract"
	"github.com/ohnishi/yahoo-news-analysis/internal/fetch"
	"github.com/ohnishi/yahoo-news-analysis/internal/jsonl"
	"github.com/ohnishi/yahoo-news-analysis/internal/model"
	"github.com/ohnishi/yahoo-news-analysis/internal/report"
)

// stubTokenizer は「〜さん」で終わる語を人名として返す。MeCab を使わずに
// json -> analysis -> markdown の連結を検証するため。
type stubTokenizer struct{}

func (stubTokenizer) Tokenize(text string) ([]analyze.Token, error) {
	var tokens []analyze.Token
	for _, w := range strings.Fields(text) {
		features := []string{"名詞", "一般", "*", "*"}
		if strings.HasSuffix(w, "さん") {
			features = []string{"名詞", "固有名詞", "人名", "一般"}
		}
		tokens = append(tokens, analyze.Token{Surface: w, Features: features})
	}
	return tokens, nil
}

func (stubTokenizer) Close() error { return nil }

const rssTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>テストフィード</title>
<link>https://example.com</link><description>test</description>
<item><title>【速報】山田さん が 受賞（写真）</title><link>https://example.com/1</link>
<pubDate>Fri, 18 Dec 2020 10:00:00 +0900</pubDate></item>
<item><title>山田さん と 鈴木さん が 対談</title><link>https://example.com/2</link>
<pubDate>Fri, 18 Dec 2020 12:00:00 +0900</pubDate></item>
<item><title>前日の記事 山田さん</title><link>https://example.com/3</link>
<pubDate>Thu, 17 Dec 2020 12:00:00 +0900</pubDate></item>
</channel></rss>`

// TestPipeline は取得済みRSSから report.md までを通しで検証する。
func TestPipeline(t *testing.T) {
	dir := t.TempDir()
	date := time.Date(2020, 12, 18, 0, 0, 0, 0, time.Local)
	dateStr := date.Format(daterange.Format)

	// fetch 段が残す状態を用意する。
	feeds := []model.RSSFeed{{ID: "rss/media/test", Name: "テスト", URL: "https://example.com/rss"}}
	if err := fetch.SaveFeeds(dir, feeds); err != nil {
		t.Fatalf("SaveFeeds: %v", err)
	}
	rssPath := filepath.Join(dir, dateStr, "rss", "media", "test")
	if err := os.MkdirAll(filepath.Dir(rssPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rssPath, []byte(rssTemplate), 0o644); err != nil {
		t.Fatal(err)
	}

	// json 段
	n, err := extract.Run(dir, dir, date, nil)
	if err != nil {
		t.Fatalf("extract.Run: %v", err)
	}
	if n != 2 {
		t.Fatalf("extracted %d articles, want 2 (the previous day must be excluded)", n)
	}

	// analysis 段
	got, err := analyze.Run(dir, dir, date, stubTokenizer{}, analyze.DefaultTop, nil)
	if err != nil {
		t.Fatalf("analyze.Run: %v", err)
	}
	if got != 2 {
		t.Fatalf("extracted %d keywords, want 2", got)
	}

	// markdown 段
	if err := report.Run(dir, dir, date); err != nil {
		t.Fatalf("report.Run: %v", err)
	}

	md, err := os.ReadFile(filepath.Join(dir, dateStr, report.FileName))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	out := string(md)

	want := []string{
		"2020/12/18 に話題になったキーワードランキング",
		"### 1位 山田さん （2記事）",
		"### 2位 鈴木さん （1記事）",
		"- [【速報】山田さん が 受賞（写真）](https://example.com/1)",
	}
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("report missing %q\n---\n%s", w, out)
		}
	}
	if strings.Contains(out, "前日の記事") {
		t.Errorf("report should not contain the previous day's article\n---\n%s", out)
	}
}

// topic.json は1行1オブジェクトのJSON Lines形式（従来と同じ）で書かれる。
func TestAnalyzeRunOutputFormat(t *testing.T) {
	dir := t.TempDir()
	date := time.Date(2020, 12, 18, 0, 0, 0, 0, time.Local)
	dateStr := date.Format(daterange.Format)

	articles := []model.Article{
		{Date: "2020-12-18T10:00:00+09:00", URL: "https://example.com/1",
			Name: "テスト", Title: "山田さん が 受賞"},
	}
	if err := jsonl.Write(filepath.Join(dir, dateStr, extract.ArticlesFileName), articles); err != nil {
		t.Fatal(err)
	}

	if _, err := analyze.Run(dir, dir, date, stubTokenizer{}, analyze.DefaultTop, nil); err != nil {
		t.Fatalf("analyze.Run: %v", err)
	}

	reports, err := jsonl.Read[model.Report](filepath.Join(dir, dateStr, analyze.ReportFileName))
	if err != nil {
		t.Fatalf("read topic.json: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("got %d reports, want 1", len(reports))
	}
	if reports[0].FormatDate != "2020/12/18" {
		t.Errorf("format_date = %q, want 2020/12/18", reports[0].FormatDate)
	}
	if len(reports[0].Items) != 1 || reports[0].Items[0].Word != "山田さん" {
		t.Errorf("items = %+v, want a single 山田さん", reports[0].Items)
	}
}

// キーワードが1件も無い日でも analysis 段は panic せず完走する。
func TestAnalyzeRunWithNoKeywords(t *testing.T) {
	dir := t.TempDir()
	date := time.Date(2020, 12, 18, 0, 0, 0, 0, time.Local)
	dateStr := date.Format(daterange.Format)

	articles := []model.Article{{URL: "https://example.com/1", Title: "人名を含まない見出し"}}
	if err := jsonl.Write(filepath.Join(dir, dateStr, extract.ArticlesFileName), articles); err != nil {
		t.Fatal(err)
	}

	n, err := analyze.Run(dir, dir, date, stubTokenizer{}, analyze.DefaultTop, nil)
	if err != nil {
		t.Fatalf("analyze.Run: %v", err)
	}
	if n != 0 {
		t.Fatalf("got %d keywords, want 0", n)
	}
}
