package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	violationCounterPrefix        = "violation_count:user:"
	violationNotifyCooldownPrefix = "violation_notify_cooldown:user:"
	userViolationBanPrefix        = "user_violation_ban:"
	violationCounterMinTTLSeconds = 60
)

// violationCounterIncrScript 使用 Lua 脚本原子性地增加计数并返回当前值
// 如果 key 不存在，则创建并设置过期时间（窗口从首次违规起算）
var violationCounterIncrScript = redis.NewScript(`
	local key = KEYS[1]
	local ttl = tonumber(ARGV[1])

	local count = redis.call('INCR', key)
	if count == 1 then
		redis.call('EXPIRE', key, ttl)
	end

	return count
`)

type violationCounterCache struct {
	rdb *redis.Client
}

// NewViolationCounterCache 创建用户违规计数器缓存实例
func NewViolationCounterCache(rdb *redis.Client) service.ViolationCounterCache {
	return &violationCounterCache{rdb: rdb}
}

// IncrementViolationCount 增加用户的违规计数，返回当前计数值
// window 是计数窗口时长，超过此时间计数器会自动重置
func (c *violationCounterCache) IncrementViolationCount(ctx context.Context, userID int64, window time.Duration) (int64, error) {
	key := fmt.Sprintf("%s%d", violationCounterPrefix, userID)

	ttlSeconds := int64(window / time.Second)
	if ttlSeconds < violationCounterMinTTLSeconds {
		ttlSeconds = violationCounterMinTTLSeconds // 最小1分钟
	}

	result, err := violationCounterIncrScript.Run(ctx, c.rdb, []string{key}, ttlSeconds).Int64()
	if err != nil {
		return 0, fmt.Errorf("increment violation count: %w", err)
	}

	return result, nil
}

// ResetViolationCount 重置用户的违规计数
func (c *violationCounterCache) ResetViolationCount(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("%s%d", violationCounterPrefix, userID)
	return c.rdb.Del(ctx, key).Err()
}

// ClaimViolationNotifyCooldown 抢占用户的违规告警冷却窗口（SET NX EX）
// 返回 true 表示获得发送权；冷却期内重复调用返回 false
func (c *violationCounterCache) ClaimViolationNotifyCooldown(ctx context.Context, userID int64, cooldown time.Duration) (bool, error) {
	if cooldown <= 0 {
		return true, nil
	}
	key := fmt.Sprintf("%s%d", violationNotifyCooldownPrefix, userID)
	ok, err := c.rdb.SetNX(ctx, key, time.Now().Unix(), cooldown).Result()
	if err != nil {
		return false, fmt.Errorf("claim violation notify cooldown: %w", err)
	}
	return ok, nil
}

// SetUserViolationBan 写入用户的临时封禁键，值为解封时间（Unix 秒），TTL 即封禁时长
// 到期后键自动过期，用户自动恢复，无需清理任务
func (c *violationCounterCache) SetUserViolationBan(ctx context.Context, userID int64, until time.Time, ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("set user violation ban: non-positive ttl")
	}
	key := fmt.Sprintf("%s%d", userViolationBanPrefix, userID)
	return c.rdb.Set(ctx, key, until.Unix(), ttl).Err()
}

// GetUserViolationBan 读取用户的临时封禁状态
// 返回 (解封时间, 是否封禁中)；键不存在、值非法或已过解封时间均视为未封禁
func (c *violationCounterCache) GetUserViolationBan(ctx context.Context, userID int64) (time.Time, bool, error) {
	key := fmt.Sprintf("%s%d", userViolationBanPrefix, userID)
	raw, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("get user violation ban: %w", err)
	}
	unix, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, false, nil
	}
	until := time.Unix(unix, 0)
	if !time.Now().Before(until) {
		return time.Time{}, false, nil
	}
	return until, true, nil
}

// ClearUserViolationBan 删除用户的临时封禁键（管理员解除封禁）
func (c *violationCounterCache) ClearUserViolationBan(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("%s%d", userViolationBanPrefix, userID)
	return c.rdb.Del(ctx, key).Err()
}
