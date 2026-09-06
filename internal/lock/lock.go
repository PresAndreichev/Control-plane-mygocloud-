package lock

import "context"

type UnlockFunc func() error

type Locker interface {
	Lock(ctx context.Context, key string) (UnlockFunc, error)
}
