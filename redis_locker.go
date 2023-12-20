package cron_with_lock

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/talentsec/go-common/cache"
	"github.com/talentsec/go-common/util"
	"strconv"
	"sync"
	"time"
)

type RedisLocker struct {
	ctx     context.Context
	rds     *redis.Client
	closers sync.Map
}

type RedisLockerConfig struct {
	Default *redis.Client
	DSN     string
}

func (this RedisLockerConfig) IsEmpty() bool {
	return this.Default == nil && this.DSN == ""
}

func NewRedisLocker(ctx context.Context, config RedisLockerConfig) (*RedisLocker, error) {
	if config.Default != nil {
		return &RedisLocker{
			ctx:     ctx,
			rds:     config.Default,
			closers: sync.Map{},
		}, nil
	}

	client, err := cache.NewRedis(&cache.Config{
		DSN: config.DSN,
	})
	if err != nil {
		return nil, err
	}
	return &RedisLocker{
		ctx: ctx,
		rds: client,
	}, nil
}

func (this *RedisLocker) LockTask(key string, expire time.Duration) (string, bool) {
	gid, _ := util.GetGoroutineId()
	value := util.InternalIp() + ":" + strconv.Itoa(int(gid))
	lockResult := this.rds.SetNX(this.ctx, key, value, expire).Val()
	if !lockResult {
		return "", lockResult
	}
	c := make(chan struct{})
	this.closers.Store(key, c)
	go this.startWatchDog(key, value, expire, c)
	return value, lockResult
}

func (this *RedisLocker) UnLockTask(key, value string) bool {
	script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
    		return redis.call("del", KEYS[1])
		else
    		return 0
		end
`)
	result := script.Run(this.ctx, this.rds, []string{key}, value).Val().(int64)
	if result <= 0 {
		return false
	}
	if this.closeWatchDog(key) {
		this.closers.Delete(key)
	}
	return true
}

func (this *RedisLocker) GetLockValue(key string) string {
	return this.rds.Get(this.ctx, key).Val()
}

func (this *RedisLocker) DelLock(key string) bool {
	result := this.rds.Del(this.ctx, key).Val()
	if result <= 0 {
		return false
	}
	if this.closeWatchDog(key) {
		this.closers.Delete(key)
	}
	return true
}

func (this *RedisLocker) ScanLocks() []string {
	result := make([]string, 0)
	this.closers.Range(func(key, value any) bool {
		result = append(result, key.(string))
		return true
	})
	return result
}

func (this *RedisLocker) closeWatchDog(key string) bool {
	c, ok := this.closers.Load(key)
	if ok {
		closer := c.(chan struct{})
		close(closer)
		return true
	}
	return false
}

func (this *RedisLocker) startWatchDog(key, value string, expire time.Duration, c chan struct{}) {
	expireInterval := time.Duration(float64(expire) * 0.8)
	ticker := time.NewTicker(expireInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:

			script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
    		return redis.call("expire", KEYS[1], ARGV[2])
		else
    		return 0
		end
`)
			script.Run(this.ctx, this.rds, []string{key}, value, expire.Seconds()).Val()
		case <-c:
			return
		}
	}
}
