package cron_with_lock

import (
	"context"
	"errors"
	"fmt"
	c "github.com/robfig/cron"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type CronConfig struct {
	RedisLockerConfig RedisLockerConfig // default nil
}

type Task struct {
	Name           string
	Spec           string
	Executor       func() interface{}
	ResultCapacity int  // the capacity of executor result
	ShouldLock     bool // use distributed lock if true
	LockExpire     int  // seconds
}

type Cron struct {
	tasks  []Task
	c      *c.Cron
	count  int
	result sync.Map
	locker TaskLocker
}

func NewCron(config ...CronConfig) (*Cron, error) {
	cron := &Cron{
		tasks:  make([]Task, 0),
		c:      c.New(),
		result: sync.Map{},
	}
	if len(config) > 0 {
		locker, err := initLocker(config[0])
		if err != nil {
			return nil, err
		}
		cron.locker = locker
	}
	return cron, nil
}

func initLocker(config CronConfig) (TaskLocker, error) {
	if !config.RedisLockerConfig.IsEmpty() {
		return NewRedisLocker(context.Background(), config.RedisLockerConfig)
	} // other locker implement
	return nil, nil
}

func (this *Cron) AddTask(task Task) {
	task.Name = this.DecorateName(task.Name)
	this.tasks = append(this.tasks, task)
}

func (this *Cron) Start() error {
	if this.c == nil {
		return errors.New("should NewCron firstly")
	}
	if err := this.taskShouldNotDuplication(); err != nil {
		return err
	}
	for _, task := range this.tasks {
		err := this.c.AddFunc(task.Spec, this.taskWithRecover(task))
		if err != nil {
			return err
		}
		this.count++
	}
	go this.c.Run()
	return nil
}

func (this *Cron) taskShouldNotDuplication() error {
	names := make(map[string]struct{})
	for _, task := range this.tasks {
		if _, exist := names[task.Name]; exist {
			return errors.New(task.Name + " duplication")
		}
		names[task.Name] = struct{}{}
	}
	return nil
}

func (this *Cron) taskWithRecover(task Task) func() {
	return func() {
		defer func() {
			if err := recover(); err != nil {
				logrus.Errorf("cron task %+v panic:%+v", task.Name, err)
			}
		}()
		var res interface{}
		if task.ShouldLock && this.locker != nil {
			value, ok := this.locker.LockTask(task.Name, time.Duration(task.LockExpire)*time.Second)
			if ok {
				res = task.Executor()
				fmt.Println(this.locker.UnLockTask(task.Name, value), value)
			}
		} else {
			res = task.Executor()
		}
		if res != nil && task.ResultCapacity > 0 {
			messages, ok := this.result.Load(task.Name)
			if !ok {
				result := []interface{}{res}
				this.result.Store(task.Name, result)
				return
			}

			v := messages.([]interface{})
			// 超出容量限制淘汰最久的消息
			if len(v) == task.ResultCapacity {
				v = v[1:]
			}
			v = append(v, res)
		}
	}
}

func (this *Cron) GetCount() int {
	return this.count
}

func (this *Cron) Stop() {
	this.c.Stop()
	this.result = sync.Map{}
	this.count = 0
	this.tasks = nil
	this.c = nil
	this.cleanLocks()
	this.locker = nil
}

func (this *Cron) cleanLocks() {
	tasks := this.ScanLockedTasks()
	for _, task := range tasks {
		this.locker.DelLock(task)
	}
}

func (this *Cron) GetResult(taskName string) []interface{} {
	value, ok := this.result.Load(this.DecorateName(taskName))
	if !ok {
		return []interface{}{}
	}
	return value.([]interface{})
}

func (this *Cron) ScanLockedTasks() []string {
	if this.locker == nil {
		return nil
	}
	return this.locker.ScanLocks()
}

func (this *Cron) DecorateName(taskName string) string {
	return "cron:" + taskName
}
