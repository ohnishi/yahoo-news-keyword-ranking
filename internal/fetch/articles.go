package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/ohnishi/yahoo-news-analysis/internal/daterange"
	"github.com/ohnishi/yahoo-news-analysis/internal/fsutil"
	"github.com/ohnishi/yahoo-news-analysis/internal/jsonl"
	"github.com/ohnishi/yahoo-news-analysis/internal/model"
)

// SaveFeeds はフィード一覧を dest/rss.jsonl へ保存する。
func SaveFeeds(dest string, feeds []model.RSSFeed) error {
	if len(feeds) == 0 {
		return errors.New("no rss feeds found")
	}
	return jsonl.Write(filepath.Join(dest, FeedListFileName), feeds)
}

// LoadFeeds は src/rss.jsonl からフィード一覧を読み込む。
func LoadFeeds(src string) ([]model.RSSFeed, error) {
	return jsonl.Read[model.RSSFeed](filepath.Join(src, FeedListFileName))
}

// Articles は各フィードのRSSを取得し dest/<YYYYmmdd>/<feedID> へ保存する。
//
// 個々のフィードの失敗では処理を止めず、警告を出して次へ進む。全件失敗した
// 場合のみエラーを返す。取得できた件数を返す。
func (c *Client) Articles(ctx context.Context, feeds []model.RSSFeed, dest string, date time.Time) (int, error) {
	destDir := filepath.Join(dest, date.Format(daterange.Format))

	saved := 0
	var errs []error
	for _, feed := range feeds {
		if err := ctx.Err(); err != nil {
			return saved, err
		}
		if err := c.fetchFeed(ctx, feed, destDir); err != nil {
			c.logger.Warn("failed to fetch RSS", "id", feed.ID, "url", feed.URL, "error", err)
			errs = append(errs, err)
			continue
		}
		saved++
	}

	if saved == 0 && len(errs) > 0 {
		return 0, fmt.Errorf("all %d feeds failed: %w", len(errs), errors.Join(errs...))
	}
	return saved, nil
}

// fetchFeed は1フィードのRSSを取得して保存する。
func (c *Client) fetchFeed(ctx context.Context, feed model.RSSFeed, destDir string) error {
	res, err := c.get(ctx, feed.URL)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	path := filepath.Join(destDir, filepath.FromSlash(feed.ID))
	return fsutil.WriteFile(path, func(w io.Writer) error {
		if _, err := io.Copy(w, res.Body); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		return nil
	})
}
