package analyze

import "testing"

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"leading heading tag", "【速報】山田太郎が受賞", "山田太郎が受賞"},
		{"trailing full-width note", "山田太郎が受賞（写真）", "山田太郎が受賞"},
		{"trailing half-width note", "山田太郎が受賞(動画)", "山田太郎が受賞"},
		{"leading square bracket tag", "[社説]山田太郎の功績", "山田太郎の功績"},
		{"leading and trailing", "【速報】山田太郎が受賞（写真）", "山田太郎が受賞"},
		{"repeated leading tags", "【速報】【独自】山田太郎が受賞", "山田太郎が受賞"},
		{"repeated trailing notes", "山田太郎が受賞（写真）(動画)", "山田太郎が受賞"},
		{"angle bracket", "山田太郎の独白〈独占〉", "山田太郎の独白"},
		{"unclosed trailing bracket", "山田太郎が受賞（写真", "山田太郎が受賞"},
		{"colon removed", "速報: 山田太郎", "速報 山田太郎"},
		{"full-width colon removed", "速報： 山田太郎", "速報 山田太郎"},
		{"lowercased", "NHK: Yamada Taro", "nhk yamada taro"},
		{"surrounding whitespace", "  山田太郎が受賞  ", "山田太郎が受賞"},
		{"tag only", "【速報】", ""},
		{"empty", "", ""},
		{"no decoration", "山田太郎が受賞", "山田太郎が受賞"},
		// 閉じ括弧を伴う括弧が文中にある場合は本文の一部として残す。
		// 以前は LastIndex("(") でそこから後ろを丸ごと捨てていた。
		{"closed bracket mid-sentence is kept", "山田太郎(29)が受賞", "山田太郎(29)が受賞"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeTitle(tt.in); got != tt.want {
				t.Errorf("NormalizeTitle(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeTitleIsIdempotent(t *testing.T) {
	inputs := []string{
		"【速報】山田太郎が受賞（写真）",
		"[社説]山田太郎の功績",
		"山田太郎が受賞（写真",
		"",
	}
	for _, in := range inputs {
		once := NormalizeTitle(in)
		if twice := NormalizeTitle(once); twice != once {
			t.Errorf("NormalizeTitle(%q): first pass %q, second pass %q", in, once, twice)
		}
	}
}
