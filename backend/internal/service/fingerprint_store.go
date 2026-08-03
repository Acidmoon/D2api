package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// 模型指纹检测的文件存储层（设计文档 §4）：
//   <data_dir>/references/<model-slug>.json      参考指纹（按模型归档，不按账号）
//   <data_dir>/audits/<yyyymmdd-hhmmss>-<id>.json  每次检测一份报告
//
// 任何文件都不写 API key；外部端点只记 base_url 与 provider。
// 列表 = 扫目录按时间倒序，无需索引文件；清理策略为手工删除。

// 参考指纹来源。
const (
	FingerprintReferenceSourceAccountSampled = "account_sampled"
	FingerprintReferenceSourceZenodoImport   = "zenodo_import" // 本期未实现，预留
)

// 检测目标类型 / 报告状态。
const (
	FingerprintTargetTypeAccount  = "account"
	FingerprintTargetTypeExternal = "external"

	FingerprintStatusRunning = "running"
	FingerprintStatusDone    = "done"
	FingerprintStatusFailed  = "failed"
)

// FingerprintReferenceCell 参考指纹中单个 cell 的分布（references 文件 cells 字段值）。
type FingerprintReferenceCell struct {
	Samples      int            `json:"samples"`      // T=1.0 成功响应数（含 invalid/refusal/empty）
	Valid        int            `json:"valid"`        // 有效样本数
	Distribution map[string]int `json:"distribution"` // 归一化答案 → 计数
	T0Answers    []string       `json:"t0_answers,omitempty"`
}

// FingerprintReference 参考指纹文件（references/<model-slug>.json）。
type FingerprintReference struct {
	Model           string                               `json:"model"`
	Source          string                               `json:"source"` // account_sampled / zenodo_import
	SourceAccountID *int64                               `json:"source_account_id,omitempty"`
	EnrolledAt      time.Time                            `json:"enrolled_at"`
	Cells           map[string]*FingerprintReferenceCell `json:"cells"`
}

// FingerprintReportTarget 报告中的被测对象（external 只记 base_url/provider，不记 API key）。
type FingerprintReportTarget struct {
	Type      string `json:"type"` // account / external
	AccountID *int64 `json:"account_id,omitempty"`
	BaseURL   string `json:"base_url,omitempty"`
	Provider  string `json:"provider,omitempty"`
	Model     string `json:"model"`
}

// FingerprintReportReference 报告中引用/现场注册的参考基准摘要。
type FingerprintReportReference struct {
	Model      string    `json:"model"`
	Source     string    `json:"source"`
	EnrolledAt time.Time `json:"enrolled_at"`
}

// FingerprintReportProgress 电池执行进度。
type FingerprintReportProgress struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

// FingerprintReportCell 报告中单个 cell 的分布摘要与得分。
type FingerprintReportCell struct {
	Task       string         `json:"task"`
	Language   string         `json:"language"`
	JSD        *float64       `json:"jsd"` // 未进 JSD 的 cell 为 null
	Valid      int            `json:"valid"`
	Invalid    int            `json:"invalid"`
	Refusal    int            `json:"refusal"`
	Empty      int            `json:"empty"`
	Excluded   string         `json:"excluded,omitempty"` // hidden_reasoning / response_caching / not_applicable / insufficient_samples
	TopAnswers map[string]int `json:"top_answers"`
	// ReferenceTopAnswers 参考侧分布摘要，便于前端直接对比展示。
	ReferenceTopAnswers map[string]int `json:"reference_top_answers,omitempty"`
	T0Answers           []string       `json:"t0_answers,omitempty"`
	// Samples 仅在 keep_raw=true 时附加（原始回答文本，供人工核查）。
	Samples []string `json:"samples,omitempty"`
}

// FingerprintReport 检测报告文件（audits/<timestamp>-<id>.json）。
type FingerprintReport struct {
	ID              string                     `json:"id"`
	Target          FingerprintReportTarget    `json:"target"`
	Reference       FingerprintReportReference `json:"reference"`
	Status          string                     `json:"status"` // running / done / failed
	Progress        FingerprintReportProgress  `json:"progress"`
	Score           *float64                   `json:"score"` // 证据不足时为 null
	Verdict         string                     `json:"verdict"`
	Band            string                     `json:"band,omitempty"` // consistent / mostly_consistent / warning / anomalous
	CellCount       int                        `json:"cell_count"`     // 进入 JSD 的有效 cell 数 k
	AvgSamples      float64                    `json:"avg_samples"`    // 有效 cell 的平均有效样本数 n
	SplitHalfJSD    *float64                   `json:"split_half_jsd"`
	T0MismatchCells int                        `json:"t0_mismatch_cells"` // T=0 答案与参考不一致的 cell 数（模型被换的即时提示）
	Flags           []string                   `json:"flags"`
	Error           string                     `json:"error,omitempty"`
	CreatedBy       int64                      `json:"created_by,omitempty"`
	CreatedAt       time.Time                  `json:"created_at"`
	DurationMs      int64                      `json:"duration_ms"`
	Cells           []*FingerprintReportCell   `json:"cells"`
}

// FingerprintAuditSummary 检测记录列表行（报告摘要，不含 cells 明细）。
type FingerprintAuditSummary struct {
	ID             string                    `json:"id"`
	Target         FingerprintReportTarget   `json:"target"`
	ReferenceModel string                    `json:"reference_model"`
	Status         string                    `json:"status"`
	Progress       FingerprintReportProgress `json:"progress"`
	Score          *float64                  `json:"score"`
	Verdict        string                    `json:"verdict"`
	Flags          []string                  `json:"flags"`
	Error          string                    `json:"error,omitempty"`
	CreatedAt      time.Time                 `json:"created_at"`
	DurationMs     int64                     `json:"duration_ms"`
}

// fingerprintStore 参考指纹与检测报告的 JSON 文件读写。
type fingerprintStore struct {
	root string // data_dir 根目录
}

func newFingerprintStore(root string) *fingerprintStore {
	return &fingerprintStore{root: root}
}

func (s *fingerprintStore) referencesDir() string { return filepath.Join(s.root, "references") }
func (s *fingerprintStore) auditsDir() string     { return filepath.Join(s.root, "audits") }

// fingerprintModelSlug 把模型名转成安全文件名：小写，非 [a-z0-9._-] 字符折叠为 -。
func fingerprintModelSlug(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	var b strings.Builder
	lastDash := false
	for _, r := range model {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-'
		if ok {
			_, _ = b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			_ = b.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "unknown"
	}
	return slug
}

// writeJSONAtomic 先写临时文件再 rename，避免半截 JSON 被列表读到。
func writeJSONAtomic(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// saveReference 写入/覆盖 references/<model-slug>.json（重注册即覆盖）。
func (s *fingerprintStore) saveReference(ref *FingerprintReference) error {
	return writeJSONAtomic(filepath.Join(s.referencesDir(), fingerprintModelSlug(ref.Model)+".json"), ref)
}

// loadReference 按模型名读取参考指纹；不存在时返回 ErrFingerprintReferenceNotFound。
func (s *fingerprintStore) loadReference(model string) (*FingerprintReference, error) {
	path := filepath.Join(s.referencesDir(), fingerprintModelSlug(model)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFingerprintReferenceNotFound
		}
		return nil, fmt.Errorf("read reference: %w", err)
	}
	var ref FingerprintReference
	if err := json.Unmarshal(data, &ref); err != nil {
		return nil, fmt.Errorf("parse reference %s: %w", path, err)
	}
	return &ref, nil
}

// listReferences 扫描 references 目录，按注册时间倒序返回。
func (s *fingerprintStore) listReferences() ([]*FingerprintReference, error) {
	entries, err := os.ReadDir(s.referencesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []*FingerprintReference{}, nil
		}
		return nil, fmt.Errorf("read references dir: %w", err)
	}
	out := make([]*FingerprintReference, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.referencesDir(), e.Name()))
		if err != nil {
			continue // 单个坏文件不拖垮列表
		}
		var ref FingerprintReference
		if err := json.Unmarshal(data, &ref); err != nil {
			continue
		}
		out = append(out, &ref)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EnrolledAt.After(out[j].EnrolledAt) })
	return out, nil
}

// auditReportFileName 报告文件名：<yyyymmdd-hhmmss>-<id>.json。
func auditReportFileName(id string, createdAt time.Time) string {
	return createdAt.UTC().Format("20060102-150405") + "-" + id + ".json"
}

// saveAuditReport 写入检测报告（完成或失败时各写一次，进行中不落盘）。
func (s *fingerprintStore) saveAuditReport(rep *FingerprintReport) error {
	return writeJSONAtomic(filepath.Join(s.auditsDir(), auditReportFileName(rep.ID, rep.CreatedAt)), rep)
}

// getAuditReport 按任务 ID 查报告文件（文件名后缀匹配）。
func (s *fingerprintStore) getAuditReport(id string) (*FingerprintReport, error) {
	if id == "" || strings.ContainsAny(id, `/\`) {
		return nil, ErrFingerprintAuditNotFound
	}
	entries, err := os.ReadDir(s.auditsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFingerprintAuditNotFound
		}
		return nil, fmt.Errorf("read audits dir: %w", err)
	}
	suffix := "-" + id + ".json"
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), suffix) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.auditsDir(), e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read audit report: %w", err)
		}
		var rep FingerprintReport
		if err := json.Unmarshal(data, &rep); err != nil {
			return nil, fmt.Errorf("parse audit report %s: %w", e.Name(), err)
		}
		return &rep, nil
	}
	return nil, ErrFingerprintAuditNotFound
}

// listAuditReports 扫描 audits 目录，按创建时间倒序返回摘要。
func (s *fingerprintStore) listAuditReports() ([]*FingerprintAuditSummary, error) {
	entries, err := os.ReadDir(s.auditsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []*FingerprintAuditSummary{}, nil
		}
		return nil, fmt.Errorf("read audits dir: %w", err)
	}
	out := make([]*FingerprintAuditSummary, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.auditsDir(), e.Name()))
		if err != nil {
			continue // 单个坏文件不拖垮列表
		}
		var rep FingerprintReport
		if err := json.Unmarshal(data, &rep); err != nil {
			continue
		}
		out = append(out, rep.summary())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

// summary 从完整报告提取列表行摘要。
func (r *FingerprintReport) summary() *FingerprintAuditSummary {
	flags := r.Flags
	if flags == nil {
		flags = []string{}
	}
	return &FingerprintAuditSummary{
		ID:             r.ID,
		Target:         r.Target,
		ReferenceModel: r.Reference.Model,
		Status:         r.Status,
		Progress:       r.Progress,
		Score:          r.Score,
		Verdict:        r.Verdict,
		Flags:          flags,
		Error:          r.Error,
		CreatedAt:      r.CreatedAt,
		DurationMs:     r.DurationMs,
	}
}
