package service

import (
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// 归一化与有效性分类，严格按 docs/MODEL_FINGERPRINT_AUDIT_CN.md §6.3（论文 §IV-B）：
//  1. Unicode NFC；2. 剥离标点引号；3. 大小写折叠；
//  4. 阿拉伯-印度数字、中文数字映射为拉丁数字；5. 取首 token；6. 颜色词 canonicalize。
//
// 有效性四分类全部记录，只有 valid 进分布：
// valid / invalid（跑题或多词）/ refusal / empty。

// 有效性分类取值。
const (
	FingerprintValidityValid   = "valid"
	FingerprintValidityInvalid = "invalid"
	FingerprintValidityRefusal = "refusal"
	FingerprintValidityEmpty   = "empty"
)

// fingerprintRefusalMarkers 拒答特征子串（大小写折叠后做 Contains 匹配）。
// 一词回答任务里出现这些片段基本可判定为拒答/安全层拦截。
//
//nolint:gochecknoglobals // 只读静态词表。
var fingerprintRefusalMarkers = []string{
	"sorry", "i can't", "i cant", "i cannot", "i'm unable", "i am unable",
	"unable to", "as an ai", "cannot assist", "can't assist", "cannot fulfill",
	"抱歉", "对不起", "很遗憾", "无法回答", "无法提供", "不能回答", "没办法",
}

// fingerprintColorAliases 中英常见颜色词 → canonical code 的映射词表。
// 颜色回答只在一词任务中出现，词表覆盖常见颜色即可（§6.3 第 6 步）。
//
//nolint:gochecknoglobals // 只读静态词表。
var fingerprintColorAliases = map[string]string{
	"red": "red", "红": "red",
	"blue": "blue", "蓝": "blue",
	"green": "green", "绿": "green",
	"yellow": "yellow", "黄": "yellow",
	"purple": "purple", "violet": "purple", "紫": "purple", "紫罗兰": "purple",
	"pink": "pink", "粉": "pink", "粉红": "pink",
	"orange": "orange", "橙": "orange", "橘": "orange",
	"black": "black", "黑": "black",
	"white": "white", "白": "white",
	"gray": "gray", "grey": "gray", "灰": "gray",
	"brown": "brown", "棕": "brown", "褐": "brown", "咖啡": "brown",
	"cyan": "cyan", "teal": "cyan", "青": "cyan",
	"gold": "gold", "golden": "gold", "金": "gold",
	"silver": "silver", "银": "silver",
	"magenta": "magenta", "品红": "magenta", "洋红": "magenta",
}

// normalizeFingerprintAnswer 归一化一次探测的原始回答。
// 返回 (canonical token, validity)；非 valid 时 token 为空串。
func normalizeFingerprintAnswer(raw string) (string, string) {
	// 1. Unicode NFC。
	text := norm.NFC.String(raw)
	text = strings.TrimSpace(text)
	if text == "" {
		return "", FingerprintValidityEmpty
	}

	// 拒答判定在剥离标点之前做：拒答句依赖完整词组（如 "I can't"）。
	folded := strings.ToLower(text)
	for _, marker := range fingerprintRefusalMarkers {
		if strings.Contains(folded, marker) {
			return "", FingerprintValidityRefusal
		}
	}

	// 2. 剥离标点引号（unicode P 类，含中英文引号/括号/句号等）。
	text = strings.Map(func(r rune) rune {
		if unicode.IsPunct(r) {
			return -1
		}
		return r
	}, text)

	// 3. 大小写折叠。
	text = strings.ToLower(text)

	// 4/5. 按空白切分 token：0 个 → empty；多个 → 多词/跑题 invalid；单个 → 取该 token。
	tokens := strings.Fields(text)
	switch len(tokens) {
	case 0:
		return "", FingerprintValidityEmpty
	case 1:
	default:
		return "", FingerprintValidityInvalid
	}

	token := tokens[0]
	// 阿拉伯-印度数字、中文数字 → 拉丁数字。
	token = fingerprintMapDigits(token)
	// 颜色词 canonicalize。
	token = fingerprintCanonicalizeColor(token)
	if token == "" {
		return "", FingerprintValidityEmpty
	}
	return token, FingerprintValidityValid
}

// fingerprintMapDigits 把 token 中的阿拉伯-印度数字与中文数字映射为拉丁数字。
// 无法整体映射时返回原 token（由调用方按普通词处理）。
func fingerprintMapDigits(token string) string {
	if token == "" {
		return token
	}
	// 已是纯拉丁数字：直接返回。
	if isAllLatinDigits(token) {
		return token
	}
	// 中文数字整体解析（含 十/百/千 单位组合，如 四十二→42、一百→100）。
	if s, ok := parseChineseNumeralToken(token); ok {
		return s
	}
	// 逐字符映射：阿拉伯-印度数字 + 单个中文数字字（如 七→7）。
	var b strings.Builder
	mapped := false
	for _, r := range token {
		switch {
		case r >= '0' && r <= '9':
			_, _ = b.WriteRune(r)
		case r >= 0x0660 && r <= 0x0669: // 阿拉伯-印度数字 U+0660–U+0669
			_, _ = b.WriteRune('0' + r - 0x0660)
			mapped = true
		case r >= 0x06F0 && r <= 0x06F9: // 扩展阿拉伯-印度数字 U+06F0–U+06F9
			_, _ = b.WriteRune('0' + r - 0x06F0)
			mapped = true
		default:
			if d, ok := fingerprintChineseDigit(r); ok {
				_ = b.WriteByte('0' + byte(d))
				mapped = true
			} else {
				_, _ = b.WriteRune(r)
			}
		}
	}
	if !mapped {
		return token
	}
	return b.String()
}

func isAllLatinDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

// fingerprintChineseDigit 单个中文数字字 → 数值；非中文数字返回 false。
func fingerprintChineseDigit(r rune) (int, bool) {
	switch r {
	case '零', '〇':
		return 0, true
	case '一':
		return 1, true
	case '二', '两':
		return 2, true
	case '三':
		return 3, true
	case '四':
		return 4, true
	case '五':
		return 5, true
	case '六':
		return 6, true
	case '七':
		return 7, true
	case '八':
		return 8, true
	case '九':
		return 9, true
	}
	return 0, false
}

// parseChineseNumeralToken 把中文数字写法（含单位组合）整体解析为拉丁数字串。
// 支持 0–9999 的常见写法（探测任务的数值域最大 100，余量充足）。
// 含无法解析的字符时返回 false。
func parseChineseNumeralToken(token string) (string, bool) {
	total := 0
	current := 0
	seen := false
	for _, r := range token {
		if d, ok := fingerprintChineseDigit(r); ok {
			// 连续两个数字字、中间无单位 → 逐位写法（如 四二=42），
			// 交给调用方的逐字符映射处理，单位解析整体放弃。
			// 注意 零/〇 是例外：它作为占位出现（一百零七=107），不视为逐位写法。
			if current != 0 && d != 0 {
				return "", false
			}
			current = d
			seen = true
			continue
		}
		unit := 0
		switch r {
		case '十':
			unit = 10
		case '百':
			unit = 100
		case '千':
			unit = 1000
		default:
			return "", false
		}
		// 「十」开头省略前导一（十=10、十五=15）。
		if current == 0 {
			current = 1
		}
		total += current * unit
		current = 0
		seen = true
	}
	if !seen {
		return "", false
	}
	total += current
	return strconv.Itoa(total), true
}

// fingerprintCanonicalizeColor 颜色词 canonicalize：中英常见颜色映射到同一 code。
// 非颜色词原样返回。中文「X色」写法先剥尾字再查表。
func fingerprintCanonicalizeColor(token string) string {
	if code, ok := fingerprintColorAliases[token]; ok {
		return code
	}
	if strings.HasSuffix(token, "色") {
		if code, ok := fingerprintColorAliases[strings.TrimSuffix(token, "色")]; ok {
			return code
		}
	}
	return token
}
