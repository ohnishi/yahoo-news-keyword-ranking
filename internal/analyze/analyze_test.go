package analyze

import (
	"errors"
	"strings"
	"testing"

	"github.com/ohnishi/yahoo-news-keyword-ranking/internal/model"
)

// fakeTokenizer は空白区切りで分割し、personNames に含まれる語だけを
// 「名詞,固有名詞,人名,一般」として返す。MeCab に依存せず Rank を検証するため。
type fakeTokenizer struct {
	personNames map[string]bool
	err         error
	closed      bool
}

func newFakeTokenizer(names ...string) *fakeTokenizer {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	return &fakeTokenizer{personNames: set}
}

func (f *fakeTokenizer) Tokenize(text string) ([]Token, error) {
	if f.err != nil {
		return nil, f.err
	}
	var tokens []Token
	for _, w := range strings.Fields(text) {
		features := []string{"名詞", "一般", "*", "*"}
		if f.personNames[w] {
			features = []string{"名詞", "固有名詞", "人名", "一般"}
		}
		tokens = append(tokens, Token{Surface: w, Features: features})
	}
	return tokens, nil
}

func (f *fakeTokenizer) Close() error {
	f.closed = true
	return nil
}

func article(title, url string) model.Article {
	return model.Article{Title: title, URL: url}
}

func TestRankCountsArticlesPerKeyword(t *testing.T) {
	tk := newFakeTokenizer("山田", "鈴木")
	articles := []model.Article{
		article("山田 が 受賞", "https://example.com/1"),
		article("山田 と 鈴木 が 対談", "https://example.com/2"),
		article("鈴木 が 引退", "https://example.com/3"),
		article("鈴木 の 記録", "https://example.com/4"),
	}

	got, err := Rank(tk, articles, 0)
	if err != nil {
		t.Fatalf("Rank: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d keywords, want 2: %+v", len(got), got)
	}
	if got[0].Word != "鈴木" || got[0].Count != 3 {
		t.Errorf("first = %s/%d, want 鈴木/3", got[0].Word, got[0].Count)
	}
	if got[1].Word != "山田" || got[1].Count != 2 {
		t.Errorf("second = %s/%d, want 山田/2", got[1].Word, got[1].Count)
	}
	if len(got[0].Articles) != got[0].Count {
		t.Errorf("Count = %d but Articles has %d entries", got[0].Count, len(got[0].Articles))
	}
}

func TestRankCountsAnArticleOncePerWord(t *testing.T) {
	tk := newFakeTokenizer("山田")
	// 同一タイトルに「山田」が2回出ても、記事としては1件。
	got, err := Rank(tk, []model.Article{article("山田 と 山田", "https://example.com/1")}, 0)
	if err != nil {
		t.Fatalf("Rank: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d keywords, want 1", len(got))
	}
	if got[0].Count != 1 {
		t.Errorf("Count = %d, want 1", got[0].Count)
	}
	if len(got[0].Articles) != 1 {
		t.Errorf("Articles has %d entries, want 1", len(got[0].Articles))
	}
}

func TestRankTiesAreOrderedByWord(t *testing.T) {
	tk := newFakeTokenizer("鈴木", "山田", "佐藤")
	articles := []model.Article{
		article("山田 と 鈴木 と 佐藤", "https://example.com/1"),
	}

	// 同数のときの並びが実行ごとに揺れないこと。
	first, err := Rank(tk, articles, 0)
	if err != nil {
		t.Fatalf("Rank: %v", err)
	}
	for i := 0; i < 20; i++ {
		got, err := Rank(tk, articles, 0)
		if err != nil {
			t.Fatalf("Rank: %v", err)
		}
		for j := range got {
			if got[j].Word != first[j].Word {
				t.Fatalf("run %d differs at %d: %s vs %s", i, j, got[j].Word, first[j].Word)
			}
		}
	}
	for i := 1; i < len(first); i++ {
		if first[i-1].Word > first[i].Word {
			t.Errorf("ties not sorted by word: %s before %s", first[i-1].Word, first[i].Word)
		}
	}
}

func TestRankTruncatesToTop(t *testing.T) {
	tk := newFakeTokenizer("a", "b", "c", "d", "e")
	got, err := Rank(tk, []model.Article{article("a b c d e", "https://example.com/1")}, 3)
	if err != nil {
		t.Fatalf("Rank: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d keywords, want 3", len(got))
	}
}

// 以前は結果を無条件に ret[:100] していたため、キーワードが少ない日に panic していた。
func TestRankWithFewerKeywordsThanTop(t *testing.T) {
	tk := newFakeTokenizer("山田")
	got, err := Rank(tk, []model.Article{article("山田 が 受賞", "https://example.com/1")}, 100)
	if err != nil {
		t.Fatalf("Rank: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d keywords, want 1", len(got))
	}
}

func TestRankWithNoArticles(t *testing.T) {
	got, err := Rank(newFakeTokenizer("山田"), nil, DefaultTop)
	if err != nil {
		t.Fatalf("Rank: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d keywords, want 0", len(got))
	}
}

func TestRankSkipsArticlesWithNoKeywords(t *testing.T) {
	got, err := Rank(newFakeTokenizer("山田"), []model.Article{article("速報 のみ", "u")}, 0)
	if err != nil {
		t.Fatalf("Rank: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d keywords, want 0", len(got))
	}
}

// 解析エラーは panic ではなくエラーとして返る。
func TestRankReturnsTokenizerError(t *testing.T) {
	sentinel := errors.New("tokenizer exploded")
	tk := newFakeTokenizer("山田")
	tk.err = sentinel

	_, err := Rank(tk, []model.Article{article("山田 が 受賞", "u")}, 0)
	if !errors.Is(err, sentinel) {
		t.Fatalf("Rank error = %v, want %v", err, sentinel)
	}
}

func TestRankNormalizesTitlesBeforeTokenizing(t *testing.T) {
	tk := newFakeTokenizer("山田")
	got, err := Rank(tk, []model.Article{article("【速報】山田 が 受賞（写真）", "u")}, 0)
	if err != nil {
		t.Fatalf("Rank: %v", err)
	}
	if len(got) != 1 || got[0].Word != "山田" {
		t.Fatalf("got %+v, want the keyword 山田", got)
	}
	// 元のタイトルはそのままレポートに載る。
	if got[0].Articles[0].Title != "【速報】山田 が 受賞（写真）" {
		t.Errorf("article title = %q, want the original title", got[0].Articles[0].Title)
	}
}

func TestTokenFeatureAccessorIsBoundsSafe(t *testing.T) {
	tk := Token{Surface: "x", Features: []string{"名詞"}}
	if tk.Feature(5) != "" {
		t.Errorf("Feature(5) = %q, want empty", tk.Feature(5))
	}
	if tk.Feature(-1) != "" {
		t.Errorf("Feature(-1) = %q, want empty", tk.Feature(-1))
	}
	// 素性が足りない形態素で panic せず、人名判定も false になること。
	if tk.IsPersonName() {
		t.Error("IsPersonName() = true, want false")
	}
}
