package lockers

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestRedisLocker_OnlyOneMachineCanLock(t *testing.T) {
	locker, err := NewRedisLocker(context.Background(), RedisLockerConfig{
		Default: redis.NewClient(&redis.Options{
			Addr: "localhost:6379",
		}),
	})
	assert.Nil(t, err)

	name := "task"

	var (
		val1, val2 string
		res1, res2 bool
	)

	// case 1
	group := sync.WaitGroup{}
	group.Add(2)
	go func() {
		val1, res1 = locker.LockTask(name, 3*time.Second)
		group.Done()
	}()

	go func() {
		val2, res2 = locker.LockTask(name, 3*time.Second)
		group.Done()
	}()

	group.Wait()

	assert.False(t, res1 && res2)
	assert.True(t, res1 || res2)

	// unlock
	if val1 != "" {
		assert.True(t, locker.UnLockTask(name, val1))
	}
	if val2 != "" {
		assert.True(t, locker.UnLockTask(name, val2))
	}

	// case 2
	val1, res1 = locker.LockTask(name, 1*time.Second)
	assert.True(t, res1)
	assert.True(t, locker.UnLockTask(name, val1))

	val2, res2 = locker.LockTask(name, 1*time.Second)
	assert.True(t, res2)
	assert.True(t, locker.UnLockTask(name, val2))

	fmt.Println("end")
}

func TestRedisLocker_ShouldExtendExpireTimeByWatchDog(t *testing.T) {
	locker, err := NewRedisLocker(context.Background(), RedisLockerConfig{
		DSN: "redis://localhost:6379/1",
	})
	assert.Nil(t, err)

	tasks := []string{"one", "two", "three", "four", "five"}
	wg := sync.WaitGroup{}
	wg.Add(len(tasks))
	task := func(key string) {
		defer wg.Done()
		// set expire 2 seconds,verify key exist after 4 seconds
		val, res := locker.LockTask(key, 2*time.Second)
		assert.True(t, res)
		assert.True(t, locker.GetLockValue(key) == val)
		// verify the watch dog working
		time.Sleep(4 * time.Second)
		assert.True(t, locker.GetLockValue(key) == val)
		locker.DelLock(key)
	}
	for _, key := range tasks {
		go task(key)
	}
	wg.Wait()
}

func TestRedisLocker_ScanLocks(t *testing.T) {
	locker, err := NewRedisLocker(context.Background(), RedisLockerConfig{
		DSN: "redis://localhost:6379/1",
	})
	assert.Nil(t, err)

	key1 := "key1"
	locker.LockTask(key1, 1*time.Second)

	key2 := "key2"
	locker.LockTask(key2, 1*time.Second)

	locks := locker.ScanLocks()

	assert.Contains(t, locks, key1)
	assert.Contains(t, locks, key2)

	locker.DelLock(key1)
	locker.DelLock(key2)

	assert.Len(t, locker.ScanLocks(), 0)
}
