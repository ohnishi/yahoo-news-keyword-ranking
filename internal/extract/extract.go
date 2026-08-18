// Package extract は取得済みのRSSファイルから、対象日に公開された記事を抜き出す。
package extract

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/ohnishi/yahoo-news-keyword-ranking/internal/daterange"
	"github.com/ohnishi/yahoo-news-keyword-ranking/internal/fetch"
	"github.com/ohnishi/yahoo-news-keyword-ranking/internal/jsonl"
	"github.com/ohnishi/yahoo-news-keyword-ranking/internal/model"
)

// ArticlesFileName は抽出結果の保存ファイル名。
const ArticlesFileName = "rss.jsonl"

// Run は src のRSSファイル群から date 当日の記事を抽出し
// dest/<YYYYmmdd>/rss.jsonl へ書き出す。書き出した件数を返す。
func Run(src, dest string, date time.Time, logger *slog.Logger) (int, error) {
	if logger == nil {
		logger = slog.Default()
	}

	feeds, err := fetch.LoadFeeds(src)
	if err != nil {
		return 0, fmt.Errorf("load rss feed list: %w", err)
	}

	dateStr := date.Format(daterange.Format)
	articles, err := Collect(feeds, filepath.Join(src, dateStr), date, logger)
	if err != nil {
		return 0, err
	}
	if len(articles) == 0 {
		logger.Warn("no articles found for date", "date", dateStr)
		return 0, nil
	}

	path := filepath.Join(dest, dateStr, ArticlesFileName)
	if err := jsonl.Write(path, articles); err != nil {
		return 0, err
	}
	return len(articles), nil
}

// Collect は fetchDir 配下のRSSファイルを解析し、date 当日に公開された記事を返す。
//
// 同一URLの記事は最初に見つかったものを採用する。出力が実行ごとに揺れないよう
// URL 昇順に整列して返す（以前は map をそのまま走査していたため行順が不定だった）。
func Collect(feeds []model.RSSFeed, fetchDir string, date time.Time, logger *slog.Logger) ([]model.Article, error) {
	dateStr := date.Format(daterange.Format)
	parser := gofeed.NewParser()

	byURL := make(map[string]model.Article)
	for _, feed := range feeds {
		path := filepath.Join(fetchDir, filepath.FromSlash(feed.ID))
		parsed, err := parseFeedFile(parser, path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				// RSSリストが更新され、その日のfetchファイルが存在しないケース。
				continue
			}
			// 1フィードの失敗で全体を止めない。
			logger.Warn("failed to read RSS file", "path", path, "error", err)
			continue
		}

		for _, item := range parsed.Items {
			if _, ok := byURL[item.Link]; ok {
				continue
			}
			published := publishedAt(item, date)
			if published.Format(daterange.Format) != dateStr {
				continue
			}
			byURL[item.Link] = model.Article{
				Date:  published.Format(time.RFC3339),
				URL:   item.Link,
				Name:  parsed.Title,
				Title: item.Title,
			}
		}
	}

	articles := make([]model.Article, 0, len(byURL))
	for _, a := range byURL {
		articles = append(articles, a)
	}
	sort.Slice(articles, func(i, j int) bool { return articles[i].URL < articles[j].URL })
	return articles, nil
}

// parseFeedFile は1つのRSSファイルを開いて解析する。
func parseFeedFile(parser *gofeed.Parser, path string) (feed *gofeed.Feed, err error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, fmt.Errorf("%s: %w", path, os.ErrNotExist)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close %s: %w", path, cerr)
		}
	}()

	feed, err = parser.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("parse RSS %s: %w", path, err)
	}
	return feed, nil
}

// publishedAt は記事の公開日時をローカルタイムで返す。
// RSSに日時が無い場合は対象日そのものを用いる。
func publishedAt(item *gofeed.Item, fallback time.Time) time.Time {
	switch {
	case item.PublishedParsed != nil:
		return item.PublishedParsed.In(time.Local)
	case item.UpdatedParsed != nil:
		return item.UpdatedParsed.In(time.Local)
	default:
		return fallback
	}
}
