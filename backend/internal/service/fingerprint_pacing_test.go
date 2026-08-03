package service

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestClampFingerprintConcurrency(t *testing.T) {
	cases := []struct{ in, want int }{
		{0, fingerprintDefaultConcurrency},
		{-3, fingerprintDefaultConcurrency},
		{1, 1},
		{2, 2},
		{16, 16},
		{17, fingerprintMaxConcurrency},
		{1000, fingerprintMaxConcurrency},
	}
	for _, tc := range cases {
		if got := clampFingerprintConcurrency(tc.in); got != tc.want {
			t.Errorf("clampFingerprintConcurrency(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestClampFingerprintIntervalMs(t *testing.T) {
	// nil（未设置）→ 默认 500ms。
	if got := clampFingerprintIntervalMs(nil); got != fingerprintDefaultIntervalMs {
		t.Fatalf("nil interval = %d, want %d", got, fingerprintDefaultIntervalMs)
	}
	cases := []struct{ in, want int }{
		{0, 0},  // 显式 0 = 不限速
		{-5, 0}, // 负值 clamp 到 0
		{250, 250},
		{60000, 60000},
		{60001, fingerprintMaxIntervalMs},
	}
	for _, tc := range cases {
		v := tc.in
		if got := clampFingerprintIntervalMs(&v); got != tc.want {
			t.Errorf("clampFingerprintIntervalMs(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestFingerprintPacerEnforcesInterval(t *testing.T) {
	pacer := &fingerprintPacer{interval: 50 * time.Millisecond}
	start := time.Now()
	// 连续 3 次预约：第 2、3 次必须各等 ≥50ms → 总耗时 ≥100ms。
	for i := 0; i < 3; i++ {
		if err := pacer.wait(context.Background()); err != nil {
			t.Fatalf("wait: %v", err)
		}
	}
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("3 次 wait 总耗时 %v，应 ≥100ms", elapsed)
	}
}

func TestFingerprintPacerConcurrentSlots(t *testing.T) {
	pacer := &fingerprintPacer{interval: 20 * time.Millisecond}
	var wg sync.WaitGroup
	starts := make([]time.Time, 6)
	// 6 个并发请求：预约时隙保证第 i 个发起时刻 ≥ i×interval。
	for i := range starts {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if err := pacer.wait(context.Background()); err == nil {
				starts[idx] = time.Now()
			}
		}(i)
	}
	wg.Wait()
	base := starts[0]
	for _, ts := range starts[1:] {
		if ts.Before(base) {
			base = ts
		}
	}
	// 最后一个时隙距离首个 ≥ 5×interval。
	latest := starts[0]
	for _, ts := range starts[1:] {
		if ts.After(latest) {
			latest = ts
		}
	}
	if got := latest.Sub(base); got < 100*time.Millisecond {
		t.Fatalf("并发时隙跨度 %v，应 ≥100ms", got)
	}
}

func TestFingerprintPacerZeroInterval(t *testing.T) {
	pacer := &fingerprintPacer{interval: 0}
	start := time.Now()
	for i := 0; i < 10; i++ {
		if err := pacer.wait(context.Background()); err != nil {
			t.Fatalf("wait: %v", err)
		}
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("interval=0 不应限速，耗时 %v", elapsed)
	}
	// nil pacer 同样直通。
	var nilPacer *fingerprintPacer
	if err := nilPacer.wait(context.Background()); err != nil {
		t.Fatalf("nil pacer wait: %v", err)
	}
}

func TestParseFingerprintRetryAfter(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   time.Duration
		ok     bool
	}{
		{"秒数", "5", 5 * time.Second, true},
		{"超过上限被截断", "300", fingerprintRetryAfterCap, true},
		{"零", "0", 0, true},
		{"负数非法", "-3", 0, false},
		{"空", "", 0, false},
		{"垃圾文本", "soon", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			if tc.header != "" {
				h.Set("Retry-After", tc.header)
			}
			got, ok := parseFingerprintRetryAfter(h)
			if ok != tc.ok || (ok && got != tc.want) {
				t.Fatalf("parseFingerprintRetryAfter(%q) = (%v, %v), want (%v, %v)", tc.header, got, ok, tc.want, tc.ok)
			}
		})
	}

	// HTTP 日期格式：未来 5 秒 → 等待 (0, 120s] 区间。
	h := http.Header{}
	h.Set("Retry-After", time.Now().Add(5*time.Second).UTC().Format(http.TimeFormat))
	got, ok := parseFingerprintRetryAfter(h)
	if !ok || got <= 0 || got > fingerprintRetryAfterCap {
		t.Fatalf("HTTP 日期解析 = (%v, %v)，应在 (0, 120s]", got, ok)
	}
	// 已过期的 HTTP 日期 → 立即重试。
	h.Set("Retry-After", time.Now().Add(-time.Minute).UTC().Format(http.TimeFormat))
	got, ok = parseFingerprintRetryAfter(h)
	if !ok || got != 0 {
		t.Fatalf("过期日期应返回 (0, true)，实际 (%v, %v)", got, ok)
	}
}

// 删除参考：model 必须过与写文件相同的 slug 化，防路径穿越。
func TestDeleteReferenceSlugSafety(t *testing.T) {
	dir := t.TempDir()
	svc := &FingerprintService{store: newFingerprintStore(dir), tasks: map[string]*fingerprintTask{}}
	accountID := int64(1)

	// 先健康注册一份参考。
	if err := svc.finishReferenceRegistration("gpt-5.4", &accountID, healthyFingerprintResults(), ""); err != nil {
		t.Fatalf("注册参考: %v", err)
	}
	// 目录外放一个哨兵文件，路径穿越若生效会误删它。
	canary := dir + "/canary.json"
	if err := os.WriteFile(canary, []byte(`{"keep":true}`), 0o600); err != nil {
		t.Fatalf("写哨兵文件: %v", err)
	}

	// 穿越形式的 model：slug 化后落在 references 目录内，找不到 → 404，哨兵不受影响。
	if err := svc.DeleteReference("../../canary"); err == nil {
		t.Fatal("删除不存在的参考应返回错误")
	} else if !strings.Contains(err.Error(), "REFERENCE_NOT_FOUND") && err != ErrFingerprintReferenceNotFound {
		t.Fatalf("应返回 ReferenceNotFound，实际: %v", err)
	}
	if _, err := os.Stat(canary); err != nil {
		t.Fatalf("路径穿越影响到了目录外文件: %v", err)
	}

	// 正常删除。
	if err := svc.DeleteReference("gpt-5.4"); err != nil {
		t.Fatalf("删除已注册参考: %v", err)
	}
	if _, err := svc.store.loadReference("gpt-5.4"); err != ErrFingerprintReferenceNotFound {
		t.Fatalf("删除后应找不到参考，实际: %v", err)
	}
	// 再删一次 → 404。
	if err := svc.DeleteReference("gpt-5.4"); err != ErrFingerprintReferenceNotFound {
		t.Fatalf("重复删除应 404，实际: %v", err)
	}
}

func TestDeleteAudit(t *testing.T) {
	dir := t.TempDir()
	svc := &FingerprintService{store: newFingerprintStore(dir), tasks: map[string]*fingerprintTask{}}

	rep := &FingerprintReport{
		ID:        "del-test-1",
		Status:    FingerprintStatusDone,
		CreatedAt: time.Now(),
		Reference: FingerprintReportReference{Model: "gpt-5.4"},
	}
	if err := svc.store.saveAuditReport(rep); err != nil {
		t.Fatalf("写报告: %v", err)
	}
	if err := svc.DeleteAudit("del-test-1"); err != nil {
		t.Fatalf("删除报告: %v", err)
	}
	if _, err := svc.store.getAuditReport("del-test-1"); err != ErrFingerprintAuditNotFound {
		t.Fatalf("删除后应找不到报告，实际: %v", err)
	}
	// 不存在 → 404。
	if err := svc.DeleteAudit("del-test-1"); err != ErrFingerprintAuditNotFound {
		t.Fatalf("重复删除应 404，实际: %v", err)
	}

	// running 中的任务拒绝删除（409）。
	running := &fingerprintTask{status: FingerprintTaskStatus{
		TaskID: "running-1",
		Kind:   FingerprintTaskKindAudit,
		Status: FingerprintStatusRunning,
	}}
	svc.tasks["running-1"] = running
	if err := svc.DeleteAudit("running-1"); err != ErrFingerprintAuditRunning {
		t.Fatalf("running 任务应拒绝删除，实际: %v", err)
	}

	// id 含路径分隔符 → 404 而非穿越。
	if err := svc.DeleteAudit("../../etc/passwd"); err != ErrFingerprintAuditNotFound {
		t.Fatalf("路径分隔符 id 应 404，实际: %v", err)
	}
}
