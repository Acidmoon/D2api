package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	violationCounterPrefix        = "violation_count:account:"
	violationNotifyCooldownPrefix = "violation_notify_cooldown:account:"
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

// NewViolationCounterCache 创建账号违规计数器缓存实例
func NewViolationCounterCache(rdb *redis.Client) service.ViolationCounterCache {
	return &violationCounterCache{rdb: rdb}
}

// IncrementViolationCount 增加账号的违规计数，返回当前计数值
// window 是计数窗口时长，超过此时间计数器会自动重置
func (c *violationCounterCache) IncrementViolationCount(ctx context.Context, accountID int64, window time.Duration) (int64, error) {
	key := fmt.Sprintf("%s%d", violationCounterPrefix, accountID)

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

// ResetViolationCount 重置账号的违规计数
func (c *violationCounterCache) ResetViolationCount(ctx context.Context, accountID int64) error {
	key := fmt.Sprintf("%s%d", violationCounterPrefix, accountID)
	return c.rdb.Del(ctx, key).Err()
}

// ClaimViolationNotifyCooldown 抢占账号的违规告警冷却窗口（SET NX EX）
// 返回 true 表示获得发送权；冷却期内重复调用返回 false
func (c *violationCounterCache) ClaimViolationNotifyCooldown(ctx context.Context, accountID int64, cooldown time.Duration) (bool, error) {
	if cooldown <= 0 {
		return true, nil
	}
	key := fmt.Sprintf("%s%d", violationNotifyCooldownPrefix, accountID)
	ok, err := c.rdb.SetNX(ctx, key, time.Now().Unix(), cooldown).Result()
	if err != nil {
		return false, fmt.Errorf("claim violation notify cooldown: %w", err)
	}
	return ok, nil
}
