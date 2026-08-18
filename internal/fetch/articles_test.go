package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ohnishi/yahoo-news-analysis/internal/model"
)

func TestArticlesSavesEachFeedUnderItsID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("body of " + r.URL.Path))
	}))
	defer srv.Close()

	feeds := []model.RSSFeed{
		{ID: "rss/media/a", URL: srv.URL + "/a"},
		{ID: "rss/media/b", URL: srv.URL + "/b"},
	}
	dir := t.TempDir()
	date := time.Date(2020, 12, 18, 0, 0, 0, 0, time.Local)

	saved, err := testClient(t, 2).Articles(context.Background(), feeds, dir, date)
	if err != nil {
		t.Fatalf("Articles: %v", err)
	}
	if saved != 2 {
		t.Fatalf("saved = %d, want 2", saved)
	}

	got, err := os.ReadFile(filepath.Join(dir, "20201218", "rss", "media", "a"))
	if err != nil {
		t.Fatalf("read saved feed: %v", err)
	}
	if string(got) != "body of /a" {
		t.Errorf("content = %q, want %q", got, "body of /a")
	}
}

// 一部のフィードが失敗しても残りは取得を続ける。
func TestArticlesContinuesPastFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	feeds := []model.RSSFeed{
		{ID: "bad", URL: srv.URL + "/bad"},
		{ID: "good", URL: srv.URL + "/good"},
	}
	dir := t.TempDir()

	saved, err := testClient(t, 1).Articles(context.Background(), feeds, dir, time.Now())
	if err != nil {
		t.Fatalf("Articles: %v", err)
	}
	if saved != 1 {
		t.Errorf("saved = %d, want 1", saved)
	}
}

// 全滅した場合はエラーにする（無言で空ディレクトリを残さない）。
func TestArticlesFailsWhenEveryFeedFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	feeds := []model.RSSFeed{{ID: "a", URL: srv.URL + "/a"}, {ID: "b", URL: srv.URL + "/b"}}

	_, err := testClient(t, 1).Articles(context.Background(), feeds, t.TempDir(), time.Now())
	if err == nil {
		t.Fatal("expected an error when every feed fails, got nil")
	}
}

func TestSaveFeedsRejectsEmptyList(t *testing.T) {
	if err := SaveFeeds(t.TempDir(), nil); err == nil {
		t.Fatal("expected an error for an empty feed list, got nil")
	}
}

func TestSaveAndLoadFeedsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := []model.RSSFeed{{ID: "rss/media/a", Name: "A", URL: "https://example.com/a.xml"}}

	if err := SaveFeeds(dir, want); err != nil {
		t.Fatalf("SaveFeeds: %v", err)
	}
	got, err := LoadFeeds(dir)
	if err != nil {
		t.Fatalf("LoadFeeds: %v", err)
	}
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
