package invalidate

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/openbao/openbao/sdk/v2/helper/salt"
	"github.com/openbao/openbao/sdk/v2/logical"
)

type LoadFunc[T any] func(ctx context.Context, storage logical.Storage) (*T, error)

type Invalidateable[T any] struct {
	x    atomic.Pointer[T]
	lock sync.Mutex // protects against concurrent update operations, reads are protected using atomics
	load LoadFunc[T]
}

func NewInvalidateable[T any](load LoadFunc[T]) Invalidateable[T] {
	return Invalidateable[T]{
		load: load,
	}
}

func NewInvalidateableSalt(location string) Invalidateable[salt.Salt] {
	return Invalidateable[salt.Salt]{
		load: func(ctx context.Context, storage logical.Storage) (*salt.Salt, error) {
			return salt.NewSalt(ctx, storage, &salt.Config{
				Location: location,
			})
		},
	}
}

func (i *Invalidateable[T]) Get(ctx context.Context, storage logical.Storage) (*T, error) {
	// optimistic cache check
	x := i.x.Load()
	if x != nil {
		return x, nil
	}

	// lock
	i.lock.Lock()
	defer i.lock.Unlock()

	// check cache again after locking
	x = i.x.Load()
	if x != nil {
		return x, nil
	}

	// load from storage
	if storage == nil {
		return nil, errors.New("storage is nil")
	}
	x, err := i.load(ctx, storage)
	if err != nil {
		return nil, err
	}
	i.x.Store(x)

	return x, nil
}

func (i *Invalidateable[T]) Update(update func(*T) (*T, error)) error {
	i.lock.Lock()
	defer i.lock.Unlock()

	x, err := update(i.x.Load())
	if err != nil {
		return err
	}

	i.x.Store(x)

	return nil
}

func (i *Invalidateable[T]) Invalidate() {
	i.lock.Lock()
	defer i.lock.Unlock()
	i.x.Store(nil)
}
