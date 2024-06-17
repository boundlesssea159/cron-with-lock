# Cron-With-Lock
This is a simple tasks controller which can limit just one node to execute task in distributed system.
## Implementation
- [x] base on Redis and provide watchdog to delay the expiration.
- [ ] base on Zookeeper
## Use
````go
// init cron with redis locker
cron, _ := NewCron(CronConfig{RedisLockerConfig: lockers.RedisLockerConfig{DSN: "redis://localhost:6379/1"}})
// add task to cron
cron.AddTask(Task{
		Name:           "test",
		Spec:           "* * * * * ?",
		Executor:      func() error{
		    fmt.Println("running")
		    time.Sleep(10 * time.Second)
		    return nil
		} ,
		ShouldLock:     true,
		LockExpire:     3,
		ResultCapacity: 1,
	})
// start cron
fmt.Println(cron.Start())
// stop cron
defer cron.Stop()