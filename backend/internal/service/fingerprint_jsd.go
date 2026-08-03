package service

import "math"

// base-2 Jensen–Shannon 散度与判定分档（纯函数）。
// 设计见 docs/MODEL_FINGERPRINT_AUDIT_CN.md §7：双方各 ≥10 个有效样本的 cell 上算
// JSD 取平均得 s，再按论文标定的操作点分档；劈半自校准用于区分「负载波动」与「模型不同」。

// 判定 verdict 取值（报告文件字段，见设计文档 §4）。
const (
	FingerprintVerdictConsistent   = "consistent"
	FingerprintVerdictWarning      = "warning"
	FingerprintVerdictAnomalous    = "anomalous"
	FingerprintVerdictInsufficient = "insufficient"
)

// 分数细分档（报告 band 字段）：consistent / mostly_consistent 都归入 verdict=consistent，
// 拆出来是为了让前端能提示「基本一致 = 可能存在跨提供商栈差异」（§7.2）。
const (
	FingerprintBandConsistent       = "consistent"
	FingerprintBandMostlyConsistent = "mostly_consistent"
	FingerprintBandWarning          = "warning"
	FingerprintBandAnomalous        = "anomalous"
)

// 判定操作点阈值（论文 165 模型、2.7 万组对照试验标定，§7.2）。
const (
	// fingerprintBandConsistentMax 同部署噪声地板（中位 0.140）上限。
	fingerprintBandConsistentMax = 0.15
	// fingerprintBandMostlyConsistentMax 跨提供商栈噪声（中位 0.227）上限。
	fingerprintBandMostlyConsistentMax = 0.30
	// fingerprintBandWarningMax 警戒档上限；≥ 此值进入 impostor 中位（0.463）区域。
	fingerprintBandWarningMax = 0.45
)

// fingerprintSplitHalfMinSamples 劈半自校准中每一半的最少样本数。
const fingerprintSplitHalfMinSamples = 5

// fingerprintJSD 计算两个稀疏计数分布的 base-2 Jensen–Shannon 散度。
// 返回 [0,1]：相同分布为 0，支撑完全不相交为 1。任一分布为空时返回 0（调用方保证非空）。
func fingerprintJSD(p, q map[string]int) float64 {
	pTotal, qTotal := 0, 0
	for _, c := range p {
		pTotal += c
	}
	for _, c := range q {
		qTotal += c
	}
	if pTotal == 0 || qTotal == 0 {
		return 0
	}
	pSum, qSum := float64(pTotal), float64(qTotal)

	// JSD = (KL(p‖m) + KL(q‖m)) / 2，m = (p+q)/2，对两侧支撑集分别稀疏迭代。
	klPM := 0.0
	for k, pc := range p {
		pp := float64(pc) / pSum
		m := (pp + float64(q[k])/qSum) / 2
		klPM += pp * math.Log2(pp/m)
	}
	klQM := 0.0
	for k, qc := range q {
		qq := float64(qc) / qSum
		m := (float64(p[k])/pSum + qq) / 2
		klQM += qq * math.Log2(qq/m)
	}
	jsd := (klPM + klQM) / 2
	// 浮点误差兜底（理论上 JSD ≥ 0）。
	if jsd < 0 {
		return 0
	}
	return jsd
}

// fingerprintMeanJSD 聚合各 cell 的 JSD：算术平均得 s。空输入返回 false（无有效 cell）。
func fingerprintMeanJSD(values []float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values)), true
}

// fingerprintSplitHalfJSD 劈半自校准：样本按索引奇偶劈成两半互算 JSD（§7.2 补充判据）。
// 任一半样本数不足 fingerprintSplitHalfMinSamples 时返回 false（证据不足，不参与平均）。
func fingerprintSplitHalfJSD(samples []string) (float64, bool) {
	odd := make(map[string]int, len(samples)/2)
	even := make(map[string]int, len(samples)/2)
	for i, s := range samples {
		if i%2 == 0 {
			even[s]++
		} else {
			odd[s]++
		}
	}
	oddN, evenN := 0, 0
	for _, c := range odd {
		oddN += c
	}
	for _, c := range even {
		evenN += c
	}
	if oddN < fingerprintSplitHalfMinSamples || evenN < fingerprintSplitHalfMinSamples {
		return 0, false
	}
	return fingerprintJSD(odd, even), true
}

// fingerprintScoreBand 按 §7.2 阈值把分数 s 分到细档。
func fingerprintScoreBand(s float64) string {
	switch {
	case s <= fingerprintBandConsistentMax:
		return FingerprintBandConsistent
	case s <= fingerprintBandMostlyConsistentMax:
		return FingerprintBandMostlyConsistent
	case s < fingerprintBandWarningMax:
		return FingerprintBandWarning
	default:
		return FingerprintBandAnomalous
	}
}

// fingerprintVerdictFor 出最终判定：k/n 未达操作点 → 证据不足；否则按细档映射 verdict。
// k 为进入 JSD 的有效 cell 数，avgN 为这些 cell 的平均有效样本数。
func fingerprintVerdictFor(s float64, k int, avgN float64) (verdict, band string) {
	if k < fingerprintMinCells || avgN < fingerprintMinAvgSamples {
		return FingerprintVerdictInsufficient, ""
	}
	band = fingerprintScoreBand(s)
	switch band {
	case FingerprintBandConsistent, FingerprintBandMostlyConsistent:
		return FingerprintVerdictConsistent, band
	case FingerprintBandWarning:
		return FingerprintVerdictWarning, band
	default:
		return FingerprintVerdictAnomalous, band
	}
}
