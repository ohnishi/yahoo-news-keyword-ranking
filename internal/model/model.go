// Package model はパイプラインの各段が受け渡すデータ構造を定義する。
//
// JSONタグは出力ファイルのフォーマットそのものなので、変更すると
// 過去に生成した rss.jsonl / topic.json が読めなくなる点に注意する。
package model

// RSSFeed は Yahoo!ニュースが提供する RSS フィード1件を表す。
// ID はフィードURLのパスから導出され、取得したRSSファイルの保存パスにも使われる。
type RSSFeed struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Article は RSS から抽出したニュース記事1件を表す。
type Article struct {
	Date  string `json:"date"`
	URL   string `json:"url"`
	Name  string `json:"name"`
	Title string `json:"title"`
}

// Report は1日分のキーワードランキングを表す。
type Report struct {
	FormatDate string    `json:"format_date"`
	Date       string    `json:"date"`
	Items      []Keyword `json:"items"`
}

// Keyword はランキング1件を表す。Count は Articles の件数と常に一致する。
type Keyword struct {
	Word     string       `json:"word"`
	Count    int          `json:"count"`
	Articles []ArticleRef `json:"articles"`
}

// ArticleRef はランキングから参照する記事を表す。
type ArticleRef struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}
