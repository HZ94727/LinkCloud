package repository

import (
	"fmt"
	"strings"
	"time"

	"gitea.com/hz/linkcloud/database"
	"github.com/redis/go-redis/v9"
)

type SecurityRepository struct{}

var tokenBucketScript = redis.NewScript(`
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_per_sec = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local data = redis.call("HMGET", key, "tokens", "ts")
local tokens = tonumber(data[1])
local ts = tonumber(data[2])

if tokens == nil then
	tokens = capacity
end

if ts == nil then
	ts = now
end

local elapsed = math.max(0, now - ts)
local refill = (elapsed * refill_per_sec) / 1000.0
tokens = math.min(capacity, tokens + refill)

local allowed = 0
local retry_ms = 0

if tokens >= 1 then
	tokens = tokens - 1
	allowed = 1
else
	if refill_per_sec > 0 then
		retry_ms = math.ceil((1 - tokens) * 1000.0 / refill_per_sec)
	else
		retry_ms = 1000
	end
end

redis.call("HMSET", key, "tokens", tokens, "ts", now)
local ttl = math.ceil((capacity / refill_per_sec) * 2000)
if ttl < 1 then
	ttl = 1
end
redis.call("PEXPIRE", key, ttl)

return {allowed, retry_ms}
`)

func NewSecurityRepository() *SecurityRepository {
	return &SecurityRepository{}
}

func normalizeRateKeyPart(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func (r *SecurityRepository) TryAcquireCaptchaCooldown(email string, duration time.Duration) (bool, error) {
	key := fmt.Sprintf("rate:captcha:%s", normalizeRateKeyPart(email))
	cmd := database.Redis.SetArgs(database.Ctx, key, 1, redis.SetArgs{
		Mode: "NX",
		TTL:  duration,
	})
	ok, err := cmd.Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	return ok == "OK", nil
}

func (r *SecurityRepository) ReleaseCaptchaCooldown(email string) error {
	key := fmt.Sprintf("rate:captcha:%s", normalizeRateKeyPart(email))
	return database.Redis.Del(database.Ctx, key).Err()
}

func (r *SecurityRepository) IsLoginLocked(ip, userName string) (bool, error) {
	key := fmt.Sprintf("lock:login:%s:%s", normalizeRateKeyPart(ip), normalizeRateKeyPart(userName))
	count, err := database.Redis.Exists(database.Ctx, key).Result()
	return count > 0, err
}

func (r *SecurityRepository) RecordLoginFailure(ip, userName string, threshold int64, window time.Duration) (bool, error) {
	ipPart := normalizeRateKeyPart(ip)
	userPart := normalizeRateKeyPart(userName)
	key := fmt.Sprintf("fail:login:%s:%s", ipPart, userPart)
	lockKey := fmt.Sprintf("lock:login:%s:%s", ipPart, userPart)

	pipe := database.Redis.TxPipeline()
	// 增加失败次数
	countCmd := pipe.Incr(database.Ctx, key)
	// 给失败计数设置到期时间
	pipe.Expire(database.Ctx, key, window)

	if _, err := pipe.Exec(database.Ctx); err != nil {
		return false, err
	}

	// 达到失败阈值，比如失败5次
	if countCmd.Val() >= threshold {
		if err := database.Redis.Set(database.Ctx, lockKey, 1, window).Err(); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func (r *SecurityRepository) ClearLoginFailures(ip, userName string) error {
	key := fmt.Sprintf("fail:login:%s:%s", normalizeRateKeyPart(ip), normalizeRateKeyPart(userName))
	return database.Redis.Del(database.Ctx, key).Err()
}

func (r *SecurityRepository) IsShortCodePasswordLocked(ip, shortCode string) (bool, error) {
	key := fmt.Sprintf("lock:short_code:password:%s:%s", normalizeRateKeyPart(ip), strings.TrimSpace(shortCode))
	count, err := database.Redis.Exists(database.Ctx, key).Result()
	return count > 0, err
}

func (r *SecurityRepository) RecordShortCodePasswordFailure(ip, shortCode string, threshold int64, window time.Duration) (bool, error) {
	key := fmt.Sprintf("fail:short_code:password:%s:%s", normalizeRateKeyPart(ip), strings.TrimSpace(shortCode))
	lockKey := fmt.Sprintf("lock:short_code:password:%s:%s", normalizeRateKeyPart(ip), strings.TrimSpace(shortCode))

	pipe := database.Redis.TxPipeline()
	countCmd := pipe.Incr(database.Ctx, key)
	pipe.Expire(database.Ctx, key, window)

	if _, err := pipe.Exec(database.Ctx); err != nil {
		return false, err
	}

	if countCmd.Val() >= threshold {
		if err := database.Redis.Set(database.Ctx, lockKey, 1, window).Err(); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func (r *SecurityRepository) ClearShortCodePasswordFailures(ip, shortCode string) error {
	key := fmt.Sprintf("fail:short_code:password:%s:%s", normalizeRateKeyPart(ip), strings.TrimSpace(shortCode))
	return database.Redis.Del(database.Ctx, key).Err()
}

func (r *SecurityRepository) IncrIPRequestCount(ip string, window time.Duration) (int64, error) {
	now := time.Now().Unix()
	key := fmt.Sprintf("rate:ip:%s:%d", strings.TrimSpace(ip), now)

	pipe := database.Redis.TxPipeline()
	countCmd := pipe.Incr(database.Ctx, key)
	pipe.Expire(database.Ctx, key, window)

	if _, err := pipe.Exec(database.Ctx); err != nil {
		return 0, err
	}

	return countCmd.Val(), nil
}

func (r *SecurityRepository) AllowTokenBucket(scope, ip string, capacity int64, refillPerSecond float64) (bool, int64, error) {
	key := fmt.Sprintf("rate:bucket:%s:%s", strings.TrimSpace(scope), strings.TrimSpace(ip))
	nowMs := time.Now().UnixMilli()

	result, err := tokenBucketScript.Run(database.Ctx, database.Redis, []string{key}, capacity, refillPerSecond, nowMs).Result()
	if err != nil {
		return false, 0, err
	}

	values, ok := result.([]interface{})
	if !ok || len(values) < 2 {
		return false, 0, fmt.Errorf("unexpected token bucket result: %#v", result)
	}

	allowed, _ := values[0].(int64)
	retryMS, _ := values[1].(int64)

	return allowed == 1, retryMS, nil
}

func (r *SecurityRepository) AllowTokenBucketByKey(scope, bucketKey string, capacity int64, refillPerSecond float64) (bool, int64, error) {
	key := fmt.Sprintf("rate:bucket:%s:%s", strings.TrimSpace(scope), strings.TrimSpace(bucketKey))
	nowMs := time.Now().UnixMilli()

	result, err := tokenBucketScript.Run(database.Ctx, database.Redis, []string{key}, capacity, refillPerSecond, nowMs).Result()
	if err != nil {
		return false, 0, err
	}

	values, ok := result.([]interface{})
	if !ok || len(values) < 2 {
		return false, 0, fmt.Errorf("unexpected token bucket result: %#v", result)
	}

	allowed, _ := values[0].(int64)
	retryMS, _ := values[1].(int64)

	return allowed == 1, retryMS, nil
}
