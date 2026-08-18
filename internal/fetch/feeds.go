package fetch

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"

	"github.com/ohnishi/yahoo-news-analysis/internal/model"
)

// FeedListURL は RSS フィード一覧ページのURL。
const FeedListURL = "https://news.yahoo.co.jp/rss"

// FeedListFileName はフィード一覧の保存ファイル名。
const FeedListFileName = "rss.jsonl"

// feedPathPrefix は一覧ページ内でRSSフィードを指すリンクのパス接頭辞。
const feedPathPrefix = "/rss/"

// Feeds は RSS フィード一覧ページを取得し、フィード定義を ID 昇順で返す。
func (c *Client) Feeds(ctx context.Context) ([]model.RSSFeed, error) {
	res, err := c.get(ctx, FeedListURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	feeds, err := parseFeedList(res.Body, res.Header.Get("Content-Type"), FeedListURL)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", FeedListURL, err)
	}
	return feeds, nil
}

// parseFeedList は一覧ページのHTMLから RSS フィード定義を抽出する。
//
// contentType とHTML内のmeta情報から文字コードを判定して UTF-8 に変換する。
// 同一フィードが複数箇所からリンクされることがあるため ID で重複排除し、
// 出力が実行ごとに揺れないよう ID 昇順に整列して返す。
func parseFeedList(r io.Reader, contentType, pageURL string) ([]model.RSSFeed, error) {
	base, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url %q: %w", pageURL, err)
	}

	decoded, err := charset.NewReader(r, contentType)
	if err != nil {
		return nil, fmt.Errorf("detect charset: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(decoded)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	seen := make(map[string]struct{})
	var feeds []model.RSSFeed
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if !strings.HasPrefix(href, feedPathPrefix) {
			return
		}
		id, ok := feedID(href)
		if !ok {
			return
		}
		if _, dup := seen[id]; dup {
			return
		}
		seen[id] = struct{}{}

		ref, err := base.Parse(href)
		if err != nil {
			return
		}
		feeds = append(feeds, model.RSSFeed{
			ID:   id,
			Name: strings.TrimSpace(s.Text()),
			URL:  ref.String(),
		})
	})

	sort.Slice(feeds, func(i, j int) bool { return feeds[i].ID < feeds[j].ID })
	return feeds, nil
}

// feedID はリンクのパスからフィードIDを導出する。
//
// 例: "/rss/media/nhk.xml" -> "rss/media/nhk"
//
// ID は取得したRSSの保存パスに使われるため、Webページ由来の値をそのまま
// filepath.Join に渡さないよう、ここでパストラバーサルを弾いておく。
func feedID(href string) (string, bool) {
	id := strings.TrimSuffix(strings.TrimPrefix(href, "/"), ".xml")
	if id == "" || id == "rss" {
		return "", false
	}
	if path.Clean(id) != id || strings.HasPrefix(id, "/") || strings.Contains(id, "..") {
		return "", false
	}
	return id, true
}
