package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ohnishi/yahoo-news-keyword-ranking/internal/analyze"
	"github.com/ohnishi/yahoo-news-keyword-ranking/internal/daterange"
	"github.com/ohnishi/yahoo-news-keyword-ranking/internal/model"
)

func sampleReport() model.Report {
	return model.Report{
		FormatDate: "2020/12/18",
		Date:       "2020-12-18T00:00:00+09:00",
		Items: []model.Keyword{
			{Word: "山田太郎", Count: 2, Articles: []model.ArticleRef{
				{Title: "山田太郎が受賞", URL: "https://example.com/1"},
				{Title: "山田太郎が語る", URL: "https://example.com/2"},
			}},
			{Word: "鈴木一郎", Count: 1, Articles: []model.ArticleRef{
				{Title: "鈴木一郎が引退", URL: "https://example.com/3"},
			}},
		},
	}
}

func TestRenderMarkdown(t *testing.T) {
	var sb strings.Builder
	if err := Render(&sb, sampleReport()); err != nil {
		t.Fatalf("Render: %v", err)
	}
	got := sb.String()

	want := []string{
		`title: "2020/12/18 に話題になったキーワードランキング"`,
		"date: 2020-12-18T00:00:00+09:00",
		"### 1位 山田太郎 （2記事）",
		"### 2位 鈴木一郎 （1記事）",
		"- [山田太郎が受賞](https://example.com/1)",
		"- [鈴木一郎が引退](https://example.com/3)",
	}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("output missing %q\n---\n%s", w, got)
		}
	}
}

// text/template を使うため、タイトル中の記号はHTMLエスケープされない。
func TestRenderDoesNotEscapeTitles(t *testing.T) {
	r := model.Report{
		FormatDate: "2020/12/18",
		Items: []model.Keyword{{Word: "a", Count: 1, Articles: []model.ArticleRef{
			{Title: `"引用" & <山括弧>`, URL: "https://example.com/1"},
		}}},
	}
	var sb strings.Builder
	if err := Render(&sb, r); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(sb.String(), `"引用" & <山括弧>`) {
		t.Errorf("title was altered:\n%s", sb.String())
	}
}

func TestRunReadsTopicJSONAndWritesMarkdown(t *testing.T) {
	dir := t.TempDir()
	date := time.Date(2020, 12, 18, 0, 0, 0, 0, time.Local)
	dateStr := date.Format(daterange.Format)

	b, err := json.Marshal(sampleReport())
	if err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(dir, dateStr, analyze.ReportFileName)
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatal(err)
	}
	// analyze は JSON Lines ライタで書くので末尾に改行が付く。それも読めること。
	if err := os.WriteFile(srcPath, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Run(dir, dir, date); err != nil {
		t.Fatalf("Run: %v", err)
	}

	out, err := os.ReadFile(filepath.Join(dir, dateStr, FileName))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(out), "### 1位 山田太郎 （2記事）") {
		t.Errorf("unexpected report:\n%s", out)
	}
}

func TestRunFailsOnEmptyRanking(t *testing.T) {
	dir := t.TempDir()
	date := time.Date(2020, 12, 18, 0, 0, 0, 0, time.Local)
	srcPath := filepath.Join(dir, date.Format(daterange.Format), analyze.ReportFileName)
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, []byte(`{"format_date":"2020/12/18","items":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Run(dir, dir, date); err == nil {
		t.Fatal("expected an error for an empty ranking, got nil")
	}
}

func TestRunFailsOnMissingSource(t *testing.T) {
	dir := t.TempDir()
	date := time.Date(2020, 12, 18, 0, 0, 0, 0, time.Local)
	if err := Run(dir, dir, date); err == nil {
		t.Fatal("expected an error for a missing topic.json, got nil")
	}
}

// front matter は1行目から始まる必要がある。先頭に空行があると
// Hugo や Jekyll が front matter として認識しない。
func TestRenderStartsWithFrontMatter(t *testing.T) {
	var sb strings.Builder
	if err := Render(&sb, sampleReport()); err != nil {
		t.Fatalf("Render: %v", err)
	}
	out := sb.String()
	if !strings.HasPrefix(out, "---\n") {
		first, _, _ := strings.Cut(out, "\n")
		t.Errorf("report starts with %q, want it to start with the front matter delimiter", first)
	}
}
