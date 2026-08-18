// Package report はキーワードランキングを Markdown に整形する。
package report

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/ohnishi/yahoo-news-analysis/internal/analyze"
	"github.com/ohnishi/yahoo-news-analysis/internal/daterange"
	"github.com/ohnishi/yahoo-news-analysis/internal/fsutil"
	"github.com/ohnishi/yahoo-news-analysis/internal/model"
)

// FileName は生成する Markdown のファイル名。
const FileName = "report.md"

const tmplText = `---
title: "{{ .FormatDate }} に話題になったキーワードランキング"
date: {{ .Date }}
---

{{ range $i, $item := .Items -}}
### {{ rank $i }}位 {{ $item.Word }} （{{ $item.Count }}記事）
{{ range $article := $item.Articles -}}
- [{{ $article.Title }}]({{ $article.URL }})
{{ end }}
{{ end }}
`

// tmpl はテンプレートの構文エラーを起動時に検出するためパッケージ初期化時に解析する。
var tmpl = template.Must(template.New("report").
	Funcs(template.FuncMap{
		"rank": func(i int) int { return i + 1 },
	}).
	Parse(tmplText))

// Run は src/<YYYYmmdd>/topic.json を読み、dest/<YYYYmmdd>/report.md を生成する。
func Run(src, dest string, date time.Time) error {
	dateStr := date.Format(daterange.Format)

	srcPath := filepath.Join(src, dateStr, analyze.ReportFileName)
	b, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", srcPath, err)
	}

	var r model.Report
	if err := json.Unmarshal(b, &r); err != nil {
		return fmt.Errorf("parse %s: %w", srcPath, err)
	}
	if len(r.Items) == 0 {
		return fmt.Errorf("%s: %w", srcPath, errors.New("no keywords to report"))
	}

	destPath := filepath.Join(dest, dateStr, FileName)
	return fsutil.WriteFile(destPath, func(w io.Writer) error {
		return Render(w, r)
	})
}

// Render はランキングを Markdown として w へ書き出す。
func Render(w io.Writer, r model.Report) error {
	if err := tmpl.Execute(w, r); err != nil {
		return fmt.Errorf("render markdown: %w", err)
	}
	return nil
}
