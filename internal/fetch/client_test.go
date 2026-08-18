package fetch

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// quietLogger はテスト出力を汚さないロガー。
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func testClient(t *testing.T, maxAttempts int) *Client {
	t.Helper()
	return NewClient(Options{
		MaxAttempts: maxAttempts,
		Backoff:     time.Millisecond,
		Logger:      quietLogger(),
	})
}

func TestGetSucceedsOnFirstAttempt(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		io.WriteString(w, "ok")
	}))
	defer srv.Close()

	res, err := testClient(t, 3).get(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer res.Body.Close()

	if got := calls.Load(); got != 1 {
		t.Errorf("server called %d times, want 1", got)
	}
}

func TestGetRetriesServerErrorsThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		io.WriteString(w, "ok")
	}))
	defer srv.Close()

	res, err := testClient(t, 3).get(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer res.Body.Close()

	if got := calls.Load(); got != 3 {
		t.Errorf("server called %d times, want 3", got)
	}
}

// maxAttempts は「総試行回数」。以前は retry++ の位置により1回多く試行していた。
func TestGetStopsAtMaxAttempts(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := testClient(t, 3).get(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if got := calls.Load(); got != 3 {
		t.Errorf("server called %d times, want exactly 3", got)
	}
	// エラーメッセージにURLが含まれること（以前は %!s(MISSING) になっていた）。
	if !strings.Contains(err.Error(), srv.URL) {
		t.Errorf("error = %v, want it to include the url %s", err, srv.URL)
	}
	if strings.Contains(err.Error(), "MISSING") {
		t.Errorf("error = %v, has an unfilled format verb", err)
	}
}

// 4xx は再試行しても変わらないので即座に諦める。
func TestGetDoesNotRetryClientErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := testClient(t, 3).get(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("server called %d times, want 1", got)
	}
}

func TestGetRetriesTooManyRequests(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		io.WriteString(w, "ok")
	}))
	defer srv.Close()

	res, err := testClient(t, 3).get(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer res.Body.Close()

	if got := calls.Load(); got != 2 {
		t.Errorf("server called %d times, want 2", got)
	}
}

func TestGetHonoursCancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := testClient(t, 3).get(ctx, srv.URL); err == nil {
		t.Fatal("expected an error from a cancelled context, got nil")
	}
}

func TestNewClientAppliesDefaults(t *testing.T) {
	c := NewClient(Options{})
	if c.maxAttempts != DefaultMaxAttempts {
		t.Errorf("maxAttempts = %d, want %d", c.maxAttempts, DefaultMaxAttempts)
	}
	if c.backoff != DefaultBackoff {
		t.Errorf("backoff = %v, want %v", c.backoff, DefaultBackoff)
	}
	if c.httpClient == nil || c.logger == nil {
		t.Error("httpClient and logger should be non-nil")
	}
}
