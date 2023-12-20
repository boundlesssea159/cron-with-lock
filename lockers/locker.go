package lockers

import "time"

type Locker interface {
	LockTask(key string, expire time.Duration) (string, bool)
	UnLockTask(key, value string) bool
	GetLockValue(key string) string
	DelLock(key string) bool
	ScanLocks() []string
}
