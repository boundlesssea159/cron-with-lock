package cron_with_lock

import (
	"cron-with-lock/lockers"
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"
)

func taskOne() interface{} {
	return "task one"
}

func taskTwo() interface{} {
	return "task two"
}

func TestCron_ShouldRunMultiJob(t *testing.T) {
	cron, _ := NewCron()
	cron.AddTask(Task{
		Name:           "taskOne",
		Spec:           "* * * * * ?",
		Executor:       taskOne,
		ResultCapacity: 1,
	})

	cron.AddTask(Task{
		Name:           "taskTwo",
		Spec:           "* * * * * ?",
		Executor:       taskTwo,
		ResultCapacity: 1,
	})

	err := cron.Start()
	defer cron.Stop()

	if err != nil {
		log.Fatal(err.Error())
	}

	time.Sleep(3 * time.Second)

	assert.Equal(t, 2, cron.GetCount())
	assert.Contains(t, cron.GetResult("taskOne"), "task one")
	assert.Contains(t, cron.GetResult("taskTwo"), "task two")
}

func TestCron_MessageResultLengthShouldNotOverCapacity(t *testing.T) {
	cron, _ := NewCron()
	cron.AddTask(Task{
		Name:           "taskOne",
		Spec:           "* * * * * ?",
		Executor:       taskOne,
		ResultCapacity: 1,
	})
	if err := cron.Start(); err != nil {
		log.Fatal(err.Error())
	}
	defer cron.Stop()
	time.Sleep(5 * time.Second)
	assert.True(t, len(cron.GetResult("taskOne")) == 1)
}

func TestCron_ShouldAppearMostOnceOfTheSameTask(t *testing.T) {
	cron, _ := NewCron()
	cron.AddTask(Task{
		Name:           "taskOne",
		Spec:           "* * * * * ?",
		Executor:       taskOne,
		ResultCapacity: 1,
	})
	cron.AddTask(Task{
		Name:           "taskOne",
		Spec:           "* * * * * ?",
		Executor:       taskOne,
		ResultCapacity: 1,
	})
	err := cron.Start()
	assert.True(t, err != nil)
	cron.Stop()
}

func TestCron_ShouldNotTerminalIfPanic(t *testing.T) {
	f := func() interface{} {
		panic("panic")
	}
	cron, _ := NewCron()
	cron.AddTask(Task{
		Name:           "f",
		Spec:           "* * * * * ?",
		Executor:       f,
		ResultCapacity: 1,
	})
	err := cron.Start()
	assert.True(t, err == nil)
	defer cron.Stop()
	time.Sleep(2 * time.Second)
	fmt.Println("success over!")
}

func TestCron_ShouldRemoveAllLockAfterStopped(t *testing.T) {
	//client := redis.NewClient(&redis.Options{
	//	Addr: "localhost:6379",
	//})
	f := func() interface{} {
		fmt.Println("running")
		time.Sleep(10 * time.Second)
		return nil
	}

	cron, _ := NewCron(CronConfig{RedisLockerConfig: lockers.RedisLockerConfig{DSN: "redis://localhost:6379/1"}})
	cron.AddTask(Task{
		Name:           "test",
		Spec:           "* * * * * ?",
		Executor:       f,
		ShouldLock:     true,
		LockExpire:     3,
		ResultCapacity: 1,
	})
	err := cron.Start()
	assert.True(t, err == nil)
	time.Sleep(1 * time.Second)
	locks := cron.ScanLockedTasks()
	assert.Contains(t, locks, cron.DecorateName("test"))
	cron.Stop()
	locks = cron.ScanLockedTasks()
	assert.Nil(t, locks)
}
