package service

import "testing"

func TestNormalizeFingerprintAnswer(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		token    string
		validity string
	}{
		// 空回答
		{"空串", "", "", FingerprintValidityEmpty},
		{"纯空白", "  \n\t ", "", FingerprintValidityEmpty},
		{"纯标点", "…。", "", FingerprintValidityEmpty},

		// 数字归一化
		{"拉丁数字带标点", "42.", "42", FingerprintValidityValid},
		{"带引号空白", "  “37”  ", "37", FingerprintValidityValid},
		{"中文数字单字", "七", "7", FingerprintValidityValid},
		{"中文数字十组合", "四十二", "42", FingerprintValidityValid},
		{"中文数字省略前导一", "十五", "15", FingerprintValidityValid},
		{"中文数字百", "一百", "100", FingerprintValidityValid},
		{"中文数字逐位", "四二", "42", FingerprintValidityValid},
		{"零", "零", "0", FingerprintValidityValid},
		{"阿拉伯-印度数字", "٤٢", "42", FingerprintValidityValid},

		// 大小写折叠
		{"大写单词", "BLUE", "blue", FingerprintValidityValid},
		{"首字母大写", "Heads", "heads", FingerprintValidityValid},

		// 颜色 canonicalize
		{"英文颜色", "Blue", "blue", FingerprintValidityValid},
		{"中文颜色", "红", "red", FingerprintValidityValid},
		{"中文颜色带色字", "蓝色", "blue", FingerprintValidityValid},
		{"grey 归一到 gray", "Grey", "gray", FingerprintValidityValid},
		{"非颜色词原样", "猫", "猫", FingerprintValidityValid},

		// 多词 → invalid
		{"多词英文", "42 is my answer", "", FingerprintValidityInvalid},
		{"多词数字", "42 43", "", FingerprintValidityInvalid},

		// 拒答
		{"英文拒答", "Sorry, I can't do that.", "", FingerprintValidityRefusal},
		{"中文拒答", "抱歉，我无法回答这个问题。", "", FingerprintValidityRefusal},

		// 单词回答
		{"动物单词", "Cat", "cat", FingerprintValidityValid},
		{"中文动物", "猫", "猫", FingerprintValidityValid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token, validity := normalizeFingerprintAnswer(tc.raw)
			if validity != tc.validity {
				t.Fatalf("normalize(%q) validity = %q, want %q", tc.raw, validity, tc.validity)
			}
			if token != tc.token {
				t.Fatalf("normalize(%q) token = %q, want %q", tc.raw, token, tc.token)
			}
		})
	}
}

func TestNormalizeFingerprintAnswerNFC(t *testing.T) {
	// NFC：组合序列（e + U+0301）归一为单码点 é。
	token, validity := normalizeFingerprintAnswer("café")
	if validity != FingerprintValidityValid {
		t.Fatalf("validity = %q, want valid", validity)
	}
	if token != "café" {
		t.Fatalf("token = %q, want NFC form %q", token, "café")
	}
}

func TestParseChineseNumeralToken(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"十", "10", true},
		{"二十", "20", true},
		{"九十九", "99", true},
		{"一百零七", "107", true},
		{"三千", "3000", true},
		{"abc", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := parseChineseNumeralToken(tc.in)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("parseChineseNumeralToken(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestGenerateProbe(t *testing.T) {
	rng := newFingerprintRNG()
	cells := fingerprintCells()
	if len(cells) != 16 {
		t.Fatalf("fingerprintCells() = %d cells, want 16", len(cells))
	}
	seen := make(map[string]struct{}, len(cells))
	for _, c := range cells {
		seen[c.Key()] = struct{}{}
		prompt := GenerateProbe(rng, c.Task, c.Language)
		if prompt == "" {
			t.Fatalf("GenerateProbe(%q, %q) returned empty prompt", c.Task, c.Language)
		}
	}
	// 未知 task / language 返回空串。
	if got := GenerateProbe(rng, "no_such_task", fingerprintLangEN); got != "" {
		t.Fatalf("unknown task should return empty, got %q", got)
	}
	if got := GenerateProbe(rng, "random_number_1_100", "fr"); got != "" {
		t.Fatalf("unknown language should return empty, got %q", got)
	}
}
