package service

import (
	"math"
	"testing"
)

func TestFingerprintJSDIdenticalDistributions(t *testing.T) {
	p := map[string]int{"42": 5, "37": 3, "7": 2}
	q := map[string]int{"42": 50, "37": 30, "7": 20}
	if got := fingerprintJSD(p, q); got != 0 {
		t.Fatalf("identical distributions: JSD = %v, want 0", got)
	}
}

func TestFingerprintJSDDisjointSupports(t *testing.T) {
	p := map[string]int{"a": 5, "b": 5}
	q := map[string]int{"c": 5, "d": 5}
	// base-2 JSD 的支撑完全不相交时恒为 1。
	if got := fingerprintJSD(p, q); got != 1 {
		t.Fatalf("disjoint supports: JSD = %v, want 1", got)
	}
}

func TestFingerprintJSDPartialOverlap(t *testing.T) {
	p := map[string]int{"a": 5, "b": 5}
	q := map[string]int{"a": 5, "c": 5}
	got := fingerprintJSD(p, q)
	if got <= 0 || got >= 1 {
		t.Fatalf("partial overlap: JSD = %v, want in (0,1)", got)
	}
	// 对称性
	if rev := fingerprintJSD(q, p); math.Abs(rev-got) > 1e-12 {
		t.Fatalf("JSD not symmetric: %v vs %v", got, rev)
	}
}

func TestFingerprintJSDEmptyInput(t *testing.T) {
	if got := fingerprintJSD(map[string]int{}, map[string]int{"a": 1}); got != 0 {
		t.Fatalf("empty distribution: JSD = %v, want 0", got)
	}
}

func TestFingerprintMeanJSD(t *testing.T) {
	if _, ok := fingerprintMeanJSD(nil); ok {
		t.Fatal("empty input should report ok=false")
	}
	got, ok := fingerprintMeanJSD([]float64{0.1, 0.2, 0.3})
	if !ok {
		t.Fatal("non-empty input should report ok=true")
	}
	if math.Abs(got-0.2) > 1e-12 {
		t.Fatalf("mean = %v, want 0.2", got)
	}
}

func TestFingerprintSplitHalfJSD(t *testing.T) {
	// 两半分布一致：JSD 应为 0（奇偶位各得 {a:2,b:2,c:1}）。
	samples := []string{"a", "a", "a", "a", "b", "b", "b", "b", "c", "c"}
	got, ok := fingerprintSplitHalfJSD(samples)
	if !ok {
		t.Fatal("10 samples split 5/5 should be ok")
	}
	if got != 0 {
		t.Fatalf("identical halves: JSD = %v, want 0", got)
	}

	// 样本不足：任一半 < 5 → false。
	if _, ok := fingerprintSplitHalfJSD([]string{"a", "b", "a", "b"}); ok {
		t.Fatal("4 samples should report ok=false")
	}

	// 两半完全不相交 → 1。
	disjoint := []string{"a", "c", "a", "c", "a", "c", "a", "c", "a", "c"}
	got, ok = fingerprintSplitHalfJSD(disjoint)
	if !ok {
		t.Fatal("10 samples should be ok")
	}
	if got != 1 {
		t.Fatalf("disjoint halves: JSD = %v, want 1", got)
	}
}

func TestFingerprintScoreBand(t *testing.T) {
	cases := []struct {
		s    float64
		want string
	}{
		{0.0, FingerprintBandConsistent},
		{0.15, FingerprintBandConsistent},
		{0.1501, FingerprintBandMostlyConsistent},
		{0.30, FingerprintBandMostlyConsistent},
		{0.3001, FingerprintBandWarning},
		{0.4499, FingerprintBandWarning},
		{0.45, FingerprintBandAnomalous},
		{0.9, FingerprintBandAnomalous},
	}
	for _, tc := range cases {
		if got := fingerprintScoreBand(tc.s); got != tc.want {
			t.Errorf("band(%v) = %q, want %q", tc.s, got, tc.want)
		}
	}
}

func TestFingerprintVerdictFor(t *testing.T) {
	cases := []struct {
		name    string
		s       float64
		k       int
		avgN    float64
		verdict string
	}{
		{"k 不足", 0.05, 7, 15, FingerprintVerdictInsufficient},
		{"n 不足", 0.05, 16, 9.5, FingerprintVerdictInsufficient},
		{"一致", 0.10, 16, 15, FingerprintVerdictConsistent},
		{"基本一致仍 consistent", 0.227, 16, 15, FingerprintVerdictConsistent},
		{"警戒", 0.35, 16, 15, FingerprintVerdictWarning},
		{"异常", 0.463, 16, 15, FingerprintVerdictAnomalous},
		{"k 边界达标", 0.10, 8, 10, FingerprintVerdictConsistent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			verdict, _ := fingerprintVerdictFor(tc.s, tc.k, tc.avgN)
			if verdict != tc.verdict {
				t.Fatalf("verdictFor(%v, %d, %v) = %q, want %q", tc.s, tc.k, tc.avgN, verdict, tc.verdict)
			}
		})
	}
}
