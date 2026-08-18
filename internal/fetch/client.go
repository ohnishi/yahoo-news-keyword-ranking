// Package fetch は Yahoo!ニュースから RSS フィード一覧と RSS ファイルを取得する。
package fetch

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// DefaultTimeout はHTTPリクエスト1回あたりの既定タイムアウト。
const DefaultTimeout = 30 * time.Second

// DefaultMaxAttempts は既定の試行回数（リトライ回数ではなく総試行回数）。
const DefaultMaxAttempts = 3

// DefaultBackoff はリトライ間隔。
const DefaultBackoff = 3 * time.Second

// Client はリトライ付きの HTTP GET を提供する。
type Client struct {
	httpClient  *http.Client
	maxAttempts int
	backoff     time.Duration
	logger      *slog.Logger
}

// Options は Client の設定を表す。ゼロ値のフィールドには既定値が使われる。
type Options struct {
	Timeout     time.Duration
	MaxAttempts int
	Backoff     time.Duration
	Logger      *slog.Logger
	// HTTPClient を指定するとテストから任意のトランスポートを差し込める。
	HTTPClient *http.Client
}

// NewClient は Options から Client を組み立てる。
func NewClient(opts Options) *Client {
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultTimeout
	}
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = DefaultMaxAttempts
	}
	if opts.Backoff <= 0 {
		opts.Backoff = DefaultBackoff
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: opts.Timeout}
	}
	return &Client{
		httpClient:  opts.HTTPClient,
		maxAttempts: opts.MaxAttempts,
		backoff:     opts.Backoff,
		logger:      opts.Logger,
	}
}

// get は url を GET し、200 のレスポンスを返す。
// 通信エラーおよび一時的とみなせるステータス(429/5xx)は maxAttempts 回まで再試行する。
// 成功時、レスポンスボディをクローズする責務は呼び出し側にある。
func (c *Client) get(ctx context.Context, url string) (*http.Response, error) {
	var lastErr error
	for attempt := 1; attempt <= c.maxAttempts; attempt++ {
		if attempt > 1 {
			c.logger.Warn("retrying request",
				"url", url, "attempt", attempt, "max_attempts", c.maxAttempts, "error", lastErr)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("build request for %s: %w", url, err)
		}

		res, err := c.httpClient.Do(req)
		switch {
		case err != nil:
			lastErr = fmt.Errorf("request %s: %w", url, err)
		case res.StatusCode == http.StatusOK:
			return res, nil
		case isRetryable(res.StatusCode):
			res.Body.Close()
			lastErr = fmt.Errorf("request %s: status code expected 200 but was %d", url, res.StatusCode)
		default:
			// 4xx などは再試行しても結果が変わらないので即座に諦める。
			res.Body.Close()
			return nil, fmt.Errorf("request %s: status code expected 200 but was %d", url, res.StatusCode)
		}
	}
	return nil, lastErr
}

// isRetryable は再試行する価値のあるステータスコードかどうかを返す。
func isRetryable(status int) bool {
	return status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}
