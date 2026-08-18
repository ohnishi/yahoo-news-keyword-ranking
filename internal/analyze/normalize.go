package analyze

import "strings"

// bracketPair は記事タイトルの前後に付く装飾の括弧の対。
type bracketPair struct{ open, close string }

// bracketPairs は除去対象の括弧。Yahoo!ニュースのタイトルは
// "【速報】…" のような接頭タグや "…（写真）" のような接尾注記を伴う。
var bracketPairs = []bracketPair{
	{"(", ")"},
	{"（", "）"},
	{"[", "]"},
	{"【", "】"},
	{"〈", "〉"},
}

// titleNoise はキーワード抽出前にタイトルから取り除く文字列。
//
// "にも" は NEologd 辞書で直前の語と結合して誤った固有名詞として
// 切り出されることがあるための回避策。
var titleNoise = []string{":", "：", "にも"}

// NormalizeTitle は形態素解析に掛ける前に記事タイトルを整形する。
//
// 小文字化・空白除去のうえ、先頭の見出しタグと末尾の注記を括弧ごと取り除き、
// 解析を乱すノイズ文字列を削除する。閉じ括弧を欠いた開き括弧（タイトルが
// 途中で切れているケース）は、そこから末尾までを落とす。
func NormalizeTitle(title string) string {
	s := strings.TrimSpace(strings.ToLower(title))
	s = stripLeadingTags(s)
	s = stripTrailingTags(s)
	for _, noise := range titleNoise {
		s = strings.ReplaceAll(s, noise, "")
	}
	return strings.TrimSpace(s)
}

// stripLeadingTags は先頭に連なる "【…】" のような見出しタグを取り除く。
func stripLeadingTags(s string) string {
	for {
		s = strings.TrimSpace(s)
		p, ok := pairOpeningAt(s)
		if !ok {
			return s
		}
		i := strings.Index(s[len(p.open):], p.close)
		if i < 0 {
			// 閉じ括弧が無いので、これ以上は判断できない。
			return s
		}
		s = s[len(p.open)+i+len(p.close):]
	}
}

// stripTrailingTags は末尾に連なる "（…）" のような注記を取り除く。
func stripTrailingTags(s string) string {
	for {
		s = strings.TrimSpace(s)
		if p, ok := pairClosingAt(s); ok {
			i := strings.LastIndex(s[:len(s)-len(p.close)], p.open)
			if i < 0 {
				return s
			}
			s = s[:i]
			continue
		}
		if i, ok := danglingOpen(s); ok {
			s = s[:i]
			continue
		}
		return s
	}
}

// pairOpeningAt は s が開き括弧で始まっていればその対を返す。
func pairOpeningAt(s string) (bracketPair, bool) {
	for _, p := range bracketPairs {
		if strings.HasPrefix(s, p.open) {
			return p, true
		}
	}
	return bracketPair{}, false
}

// pairClosingAt は s が閉じ括弧で終わっていればその対を返す。
func pairClosingAt(s string) (bracketPair, bool) {
	for _, p := range bracketPairs {
		if strings.HasSuffix(s, p.close) {
			return p, true
		}
	}
	return bracketPair{}, false
}

// danglingOpen は閉じ括弧を伴わない開き括弧のうち、最も後ろの位置を返す。
func danglingOpen(s string) (int, bool) {
	found := -1
	for _, p := range bracketPairs {
		i := strings.LastIndex(s, p.open)
		if i < 0 || i <= found {
			continue
		}
		if strings.Contains(s[i+len(p.open):], p.close) {
			continue
		}
		found = i
	}
	if found < 0 {
		return 0, false
	}
	return found, true
}
