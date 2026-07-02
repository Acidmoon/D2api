package service

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"
)

const groupUnavailableAlertCooldown = 30 * time.Minute

type GroupUnavailableAlertInput struct {
	GroupID        int64
	GroupName      string
	Platform       string
	RequestedModel string
	Stage          string
	TotalAccounts  int
	CandidateCount int
	TopK           int
	Reason         string
	OccurredAt     time.Time
}

// GroupUnavailableAlertService sends email alerts when a configured group has
// accounts but none can be selected for an actual gateway request.
type GroupUnavailableAlertService struct {
	emailService *EmailService
	settingRepo  SettingRepository
}

func NewGroupUnavailableAlertService(emailService *EmailService, settingRepo SettingRepository) *GroupUnavailableAlertService {
	return &GroupUnavailableAlertService{
		emailService: emailService,
		settingRepo:  settingRepo,
	}
}

func (s *GroupUnavailableAlertService) NotifyAsync(input GroupUnavailableAlertInput) {
	if s == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := s.Notify(ctx, input); err != nil {
			slog.Warn("group_unavailable_alert.notify_failed",
				"group_id", input.GroupID,
				"stage", input.Stage,
				"err", err,
			)
		}
	}()
}

func (s *GroupUnavailableAlertService) Notify(ctx context.Context, input GroupUnavailableAlertInput) error {
	if s == nil || s.emailService == nil || s.settingRepo == nil {
		return nil
	}
	if input.GroupID <= 0 {
		return nil
	}
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now()
	}
	recipients := s.recipients(ctx)
	if len(recipients) == 0 {
		return nil
	}
	if !s.claimCooldown(ctx, input.GroupID, input.OccurredAt) {
		return nil
	}

	subject := fmt.Sprintf("[D2api] 分组不可用警告：%s", firstNonEmpty(strings.TrimSpace(input.GroupName), fmt.Sprintf("Group #%d", input.GroupID)))
	body := buildGroupUnavailableAlertBody(input)
	for _, recipient := range recipients {
		if err := s.emailService.SendEmail(ctx, recipient, subject, body); err != nil {
			return err
		}
	}
	return nil
}

func (s *GroupUnavailableAlertService) recipients(ctx context.Context) []string {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyGroupUnavailableAlertEmails)
	if err != nil || strings.TrimSpace(raw) == "" || raw == "[]" {
		return nil
	}
	return filterVerifiedEmails(ParseNotifyEmails(raw))
}

func (s *GroupUnavailableAlertService) claimCooldown(ctx context.Context, groupID int64, now time.Time) bool {
	raw, _ := s.settingRepo.GetValue(ctx, SettingKeyGroupUnavailableAlertCooldowns)
	cooldowns := map[string]string{}
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &cooldowns)
	}

	key := fmt.Sprintf("%d", groupID)
	if lastRaw := strings.TrimSpace(cooldowns[key]); lastRaw != "" {
		if lastSent, err := time.Parse(time.RFC3339Nano, lastRaw); err == nil && now.Sub(lastSent) < groupUnavailableAlertCooldown {
			return false
		}
	}

	cutoff := now.Add(-24 * time.Hour)
	for k, v := range cooldowns {
		ts, err := time.Parse(time.RFC3339Nano, v)
		if err != nil || ts.Before(cutoff) {
			delete(cooldowns, k)
		}
	}
	cooldowns[key] = now.Format(time.RFC3339Nano)
	data, err := json.Marshal(cooldowns)
	if err != nil {
		return true
	}
	if err := s.settingRepo.Set(ctx, SettingKeyGroupUnavailableAlertCooldowns, string(data)); err != nil {
		slog.Warn("group_unavailable_alert.cooldown_write_failed", "group_id", groupID, "err", err)
	}
	return true
}

func buildGroupUnavailableAlertBody(input GroupUnavailableAlertInput) string {
	when := input.OccurredAt.Format(time.RFC3339)
	rows := []string{
		tableRow("分组", fmt.Sprintf("%s (#%d)", firstNonEmpty(input.GroupName, "未命名分组"), input.GroupID)),
		tableRow("平台", input.Platform),
		tableRow("请求模型", firstNonEmpty(input.RequestedModel, "未指定")),
		tableRow("触发阶段", input.Stage),
		tableRow("失败原因", input.Reason),
		tableRow("分组账号数", fmt.Sprintf("%d", input.TotalAccounts)),
		tableRow("候选账号数", fmt.Sprintf("%d", input.CandidateCount)),
		tableRow("TopK", fmt.Sprintf("%d", input.TopK)),
		tableRow("触发时间", when),
	}
	return `<!doctype html><html><body style="font-family:Arial,sans-serif;line-height:1.5;color:#111827;">` +
		`<h2>D2api 分组不可用警告</h2>` +
		`<p>该分组在一次真实请求调度中没有可用账号。邮件使用系统 SMTP 配置发送。</p>` +
		`<table cellpadding="6" cellspacing="0" border="1" style="border-collapse:collapse;border-color:#d1d5db;">` +
		strings.Join(rows, "") +
		`</table></body></html>`
}

func tableRow(label, value string) string {
	return fmt.Sprintf(
		`<tr><th align="left" style="background:#f3f4f6;">%s</th><td>%s</td></tr>`,
		html.EscapeString(label),
		html.EscapeString(firstNonEmpty(value, "-")),
	)
}
