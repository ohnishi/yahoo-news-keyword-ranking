// Package analyze は記事タイトルからキーワードを抽出してランク付けする。
//
// 形態素解析器は Tokenizer インタフェースの裏に隠してあるため、
// このパッケージ自体は MeCab (cgo) に依存せずテストできる。
package analyze

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"time"

	"github.com/ohnishi/yahoo-news-analysis/internal/daterange"
	"github.com/ohnishi/yahoo-news-analysis/internal/extract"
	"github.com/ohnishi/yahoo-news-analysis/internal/jsonl"
	"github.com/ohnishi/yahoo-news-analysis/internal/model"
)

// ReportFileName はランキング結果の保存ファイル名。
const ReportFileName = "topic.json"

// DefaultTop は既定で残すキーワード件数。
const DefaultTop = 30

// Token は形態素1件を表す。
type Token struct {
	// Surface は表層形。
	Surface string
	// Features は素性（品詞, 品詞細分類1, ... ）をカンマで分割したもの。
	Features []string
}

// Feature は i 番目の素性を返す。範囲外なら空文字を返す。
func (t Token) Feature(i int) string {
	if i < 0 || i >= len(t.Features) {
		return ""
	}
	return t.Features[i]
}

// IsPersonName は「名詞,固有名詞,人名,一般」の形態素かどうかを返す。
func (t Token) IsPersonName() bool {
	return t.Feature(0) == "名詞" &&
		t.Feature(1) == "固有名詞" &&
		t.Feature(2) == "人名" &&
		t.Feature(3) == "一般"
}

// Tokenizer は文を形態素へ分割する。
type Tokenizer interface {
	Tokenize(text string) ([]Token, error)
	Close() error
}

// Run は src/<YYYYmmdd>/rss.jsonl を解析し、
// dest/<YYYYmmdd>/topic.json へランキングを書き出す。抽出件数を返す。
func Run(src, dest string, date time.Time, tk Tokenizer, top int, logger *slog.Logger) (int, error) {
	if logger == nil {
		logger = slog.Default()
	}

	dateStr := date.Format(daterange.Format)
	path := filepath.Join(src, dateStr, extract.ArticlesFileName)
	articles, err := jsonl.Read[model.Article](path)
	if err != nil {
		return 0, err
	}
	if len(articles) == 0 {
		logger.Warn("no articles to analyze", "path", path)
	}

	keywords, err := Rank(tk, articles, top)
	if err != nil {
		return 0, err
	}

	report := model.Report{
		FormatDate: date.Format("2006/01/02"),
		Date:       date.Format(time.RFC3339),
		Items:      keywords,
	}
	// topic.json は1オブジェクトのみだが、従来と同じバイト列になるよう
	// JSON Lines ライタで書き出す。
	if err := jsonl.Write(filepath.Join(dest, dateStr, ReportFileName), []model.Report{report}); err != nil {
		return 0, err
	}
	return len(keywords), nil
}

// Rank は記事タイトルから人名キーワードを抽出し、登場記事数の多い順に
// 上位 top 件を返す。top が 0 以下なら全件返す。
//
// 同一タイトル内に同じ語が複数回現れても、その記事は1回だけ数える。
func Rank(tk Tokenizer, articles []model.Article, top int) ([]model.Keyword, error) {
	byWord := make(map[string][]model.ArticleRef)

	for _, article := range articles {
		title := NormalizeTitle(article.Title)
		if title == "" {
			continue
		}
		tokens, err := tk.Tokenize(title)
		if err != nil {
			return nil, fmt.Errorf("tokenize %q: %w", article.Title, err)
		}

		seen := make(map[string]struct{})
		for _, token := range tokens {
			if !token.IsPersonName() {
				continue
			}
			word := token.Surface
			if _, dup := seen[word]; dup {
				continue
			}
			seen[word] = struct{}{}
			byWord[word] = append(byWord[word], model.ArticleRef{
				Title: article.Title,
				URL:   article.URL,
			})
		}
	}

	keywords := make([]model.Keyword, 0, len(byWord))
	for word, refs := range byWord {
		keywords = append(keywords, model.Keyword{
			Word:     word,
			Count:    len(refs),
			Articles: refs,
		})
	}

	// 件数降順。同数のときは語順で固定し、実行ごとに順序が揺れないようにする。
	sort.Slice(keywords, func(i, j int) bool {
		if keywords[i].Count != keywords[j].Count {
			return keywords[i].Count > keywords[j].Count
		}
		return keywords[i].Word < keywords[j].Word
	})

	if top > 0 && len(keywords) > top {
		keywords = keywords[:top]
	}
	return keywords, nil
}
