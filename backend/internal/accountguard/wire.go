package accountguard

import (
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/wire"
)

// ProvideGuardEvaluator 创建账号违规守护专用的同步评估器。
// 不传事件落库 repo 与指标：守护路径的违规计数由 GuardService 自己维护，
// 避免与 prompt-audit 的异步/同步审计事件重复记录。
func ProvideGuardEvaluator(scanner *securityaudit.OpenAICompatibleScanner) *securityaudit.GuardEvaluator {
	return securityaudit.NewGuardEvaluator(scanner, nil, nil)
}

// ProviderSet 是 accountguard 包的 Wire provider 集合。
var ProviderSet = wire.NewSet(
	ProvideGuardEvaluator,
	NewGuardService,
	wire.Bind(new(service.AccountViolationGuard), new(*GuardService)),
	wire.Bind(new(activeConfigStore), new(*securityaudit.ConfigManager)),
	wire.Bind(new(promptGuardEvaluator), new(*securityaudit.GuardEvaluator)),
	wire.Bind(new(AccountBanRepository), new(service.AccountRepository)),
	wire.Bind(new(SettingValueReader), new(service.SettingRepository)),
	wire.Bind(new(EmailSender), new(*service.EmailService)),
)
