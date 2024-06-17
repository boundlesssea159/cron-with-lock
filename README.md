# Cron-With-Lock
This is a simple tasks controller which can limit just one node to execute task in distributed system.
## Implementation
- [x] base on Redis and provide watch-dog to delay the expiration.
- [ ] base on Zookeeper
## Use
````go
// init cron with redis locker
cron, _ := NewCron(CronConfig{RedisLockerConfig: lockers.RedisLockerConfig{DSN: "redis://localhost:6379/1"}})
// add task to cron
cron.AddTask(Task{
		Name:           "test",         // task name
		Spec:           "* * * * * ?", // task cron spec
		Executor:      func() error{   // task logic
		    fmt.Println("running")
		    time.Sleep(10 * time.Second)
		    return nil
		} ,
		ShouldLock:     true, // decide whether the task need to be locked
		LockExpire:     3, // lock duration
		ResultCapacity: 1, // record task result and result num is limited by this value
	})
// start cron
fmt.Println(cron.Start())
// stop cron
defer cron.Stop()