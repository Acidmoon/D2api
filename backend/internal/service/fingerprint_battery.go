package service

import (
	"math/rand/v2"
	"time"
)

// 模型指纹检测：探测电池定义。
// 设计见 docs/MODEL_FINGERPRINT_AUDIT_CN.md §6：8 个任务 × 中英 2 语言 = 16 个 cell，
// 每个 cell 在 T=1.0 下采样 15 次，另附 2 次 T=0 作确定性参考。
//
// 每个 (task, language) 维护一个人工改写的释义池，探测时随机抽题，
// 防上游识别固定审计串并特殊处理（论文 T2 对手）。

const (
	// fingerprintSamplesPerCell 每 cell 的 T=1.0 采样次数（论文 k=16 操作点配套）。
	fingerprintSamplesPerCell = 15
	// fingerprintGreedySamplesPerCell 每 cell 追加的 T=0 确定性参考次数。
	fingerprintGreedySamplesPerCell = 2
	// fingerprintMaxTokens 单次探测的输出 token 上限（论文上限）。
	fingerprintMaxTokens = 16
	// fingerprintProbeTemperature 指纹定义所在的采样温度。
	fingerprintProbeTemperature = 1.0
	// fingerprintGreedyTemperature T=0 确定性参考的采样温度。
	fingerprintGreedyTemperature = 0.0
	// fingerprintSystemPrompt 强制一词回答的系统提示。
	fingerprintSystemPrompt = "Answer with a single word or number only."
	// fingerprintMinValidSamples cell 进入 JSD 计算所需的双侧最小有效样本数（§7.1）。
	fingerprintMinValidSamples = 10
	// fingerprintMinCells 出判定所需的最少有效 cell 数 k（§7.2）。
	fingerprintMinCells = 8
	// fingerprintMinAvgSamples 出判定所需的每 cell 平均有效样本数 n（§7.2）。
	fingerprintMinAvgSamples = 10.0

	// fingerprintProbeTimeout 单次探测请求总超时（与监控模块一致）。
	fingerprintProbeTimeout = 45 * time.Second
	// fingerprintMaxRetries 单探测失败后的指数退避重试次数。
	fingerprintMaxRetries = 2
	// fingerprintRetryBaseBackoff 重试退避基数（第 n 次重试等待 base * 2^n）。
	fingerprintRetryBaseBackoff = 500 * time.Millisecond
	// fingerprintWorkerCount 电池执行的并发 worker 数（有界并发 4–8 区间）。
	fingerprintWorkerCount = 6
)

// 电池覆盖的语言。
const (
	fingerprintLangEN = "en"
	fingerprintLangZH = "zh"
)

// fingerprintTaskDef 描述一个探测任务定义：稳定 ID + 各语言释义池。
type fingerprintTaskDef struct {
	id      string
	prompts map[string][]string // language -> 释义池
}

//nolint:gochecknoglobals // 电池定义是只读静态数据，初始化后不变更。
var fingerprintTasks = []fingerprintTaskDef{
	{
		id: "random_number_1_100",
		prompts: map[string][]string{
			fingerprintLangEN: {
				"Name a random number between 1 and 100.",
				"Pick a random integer from 1 to 100.",
				"Give me any random number between one and a hundred.",
				"Choose a random number from 1 to 100.",
				"Tell me a random number between 1 and 100.",
			},
			fingerprintLangZH: {
				"说一个 1 到 100 之间的随机数。",
				"随机给我一个 1 到 100 的整数。",
				"从 1 到 100 中随便挑一个数。",
				"请随机想一个 1 到 100 之间的数字。",
				"给我一个 1 到 100 之间的任意数字。",
			},
		},
	},
	{
		id: "random_number_1_10",
		prompts: map[string][]string{
			fingerprintLangEN: {
				"Name a random number between 1 and 10.",
				"Pick a random integer from 1 to 10.",
				"Give me any random number between one and ten.",
				"Choose a random number from 1 to 10.",
				"Tell me a random number between 1 and 10.",
			},
			fingerprintLangZH: {
				"说一个 1 到 10 之间的随机数。",
				"随机给我一个 1 到 10 的整数。",
				"从 1 到 10 中随便挑一个数。",
				"请随机想一个 1 到 10 之间的数字。",
				"给我一个 1 到 10 之间的任意数字。",
			},
		},
	},
	{
		id: "favorite_number",
		prompts: map[string][]string{
			fingerprintLangEN: {
				"What is your favorite number?",
				"Tell me your favorite number.",
				"Which number do you like the most?",
				"Name your favorite number.",
				"What number is your favorite?",
			},
			fingerprintLangZH: {
				"你最喜欢的数字是什么？",
				"告诉我你最喜欢的数字。",
				"你最喜欢哪个数字？",
				"说一个你最喜欢的数字。",
				"你最偏爱哪个数？",
			},
		},
	},
	{
		id: "random_letter",
		prompts: map[string][]string{
			fingerprintLangEN: {
				"Name a random letter of the alphabet.",
				"Pick a random letter.",
				"Give me any random letter from A to Z.",
				"Choose a random letter of the English alphabet.",
				"Tell me a random letter.",
			},
			fingerprintLangZH: {
				"随机说一个英文字母。",
				"从 A 到 Z 中随便挑一个字母。",
				"给我一个随机字母。",
				"请随机想一个英文字母。",
				"任意说一个字母。",
			},
		},
	},
	{
		id: "random_color",
		prompts: map[string][]string{
			fingerprintLangEN: {
				"Name a random color.",
				"Pick a random color.",
				"Give me any random color.",
				"Choose a color at random.",
				"Tell me a random color.",
			},
			fingerprintLangZH: {
				"随机说一种颜色。",
				"随便给我一个颜色。",
				"请随机想一种颜色。",
				"任意说一种颜色。",
				"随机挑一种颜色告诉我。",
			},
		},
	},
	{
		id: "favorite_color",
		prompts: map[string][]string{
			fingerprintLangEN: {
				"What is your favorite color?",
				"Tell me your favorite color.",
				"Which color do you like the most?",
				"Name your favorite color.",
				"What color is your favorite?",
			},
			fingerprintLangZH: {
				"你最喜欢什么颜色？",
				"告诉我你最喜欢的颜色。",
				"你最喜欢哪种颜色？",
				"说一个你最喜欢的颜色。",
				"你最偏爱什么颜色？",
			},
		},
	},
	{
		id: "random_animal",
		prompts: map[string][]string{
			fingerprintLangEN: {
				"Name a random animal.",
				"Pick a random animal.",
				"Give me any random animal.",
				"Choose an animal at random.",
				"Tell me a random animal.",
			},
			fingerprintLangZH: {
				"随机说一种动物。",
				"随便给我一个动物名称。",
				"请随机想一种动物。",
				"任意说一种动物。",
				"随机挑一种动物告诉我。",
			},
		},
	},
	{
		id: "coin_flip",
		prompts: map[string][]string{
			fingerprintLangEN: {
				"Flip a coin. Heads or tails?",
				"I just flipped a coin. Call it: heads or tails?",
				"Simulate a coin flip. What's the result?",
				"Toss a coin and tell me the outcome.",
				"Pick one: heads or tails?",
			},
			fingerprintLangZH: {
				"抛一枚硬币，正面还是反面？",
				"模拟抛一次硬币，结果是什么？",
				"我抛了一枚硬币，你猜正面还是反面？",
				"抛一次硬币并告诉我结果。",
				"二选一：正面还是反面？",
			},
		},
	},
}

// fingerprintCell 是电池的最小单元：(task, language)。
type fingerprintCell struct {
	Task     string `json:"task"`
	Language string `json:"language"`
}

// Key 返回 cell 的稳定字符串键（"task|language"），用作参考指纹/报告 JSON 的字段名。
func (c fingerprintCell) Key() string {
	return c.Task + "|" + c.Language
}

// fingerprintCells 返回全部 16 个 cell（顺序固定：任务定义序 × en/zh）。
func fingerprintCells() []fingerprintCell {
	cells := make([]fingerprintCell, 0, len(fingerprintTasks)*2)
	for _, t := range fingerprintTasks {
		cells = append(cells, fingerprintCell{Task: t.id, Language: fingerprintLangEN})
		cells = append(cells, fingerprintCell{Task: t.id, Language: fingerprintLangZH})
	}
	return cells
}

// fingerprintShuffledCells 返回随机打乱的 cell 执行顺序（seeded shuffle，
// 避免固定的审计顺序被上游识别）。
func fingerprintShuffledCells(rng *rand.Rand) []fingerprintCell {
	cells := fingerprintCells()
	rng.Shuffle(len(cells), func(i, j int) { cells[i], cells[j] = cells[j], cells[i] })
	return cells
}

// newFingerprintRNG 创建电池专用的随机源（rand/v2 全局源已随机播种，直接取两次做种子）。
func newFingerprintRNG() *rand.Rand {
	return rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
}

// GenerateProbe 从指定 (task, language) 的释义池中随机抽一条探测 prompt。
// 未知 task 或 language 返回空串。
func GenerateProbe(rng *rand.Rand, taskID, language string) string {
	for _, t := range fingerprintTasks {
		if t.id != taskID {
			continue
		}
		pool := t.prompts[language]
		if len(pool) == 0 {
			return ""
		}
		return pool[rng.IntN(len(pool))]
	}
	return ""
}
