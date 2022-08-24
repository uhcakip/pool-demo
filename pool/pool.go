package pool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Pool[T any] interface {
	// Acquire This creates or returns a ready-to-use item from the resource pool
	Acquire(context.Context) (T, error)

	// Release This releases an active resource back to the resource pool
	Release(T)

	// NumIdle This returns the number of idle items
	NumIdle() int
}

type CreatorFunc[T any] func(context.Context) (T, error)

type idleResource[T any] struct {
	sweepTime time.Time
	value     T
}

type ResourcePool[T any] struct {
	creator        CreatorFunc[T]
	maxIdleSize    int
	maxIdleTime    time.Duration
	idleResourceCh chan idleResource[T]
	acquiredSizeCh chan bool
	locker         sync.Mutex
}

// New
// creator is a function called by the pool to create a resource
// maxIdleSize is the number of maximum idle items kept in the pool
// maxIdleTime is the maximum idle time for an idle item to be swept from the pool
func New[T any](creator CreatorFunc[T], maxIdleSize int, maxIdleTime time.Duration) Pool[T] {
	rp := &ResourcePool[T]{
		creator:        creator,
		maxIdleSize:    maxIdleSize,
		maxIdleTime:    maxIdleTime,
		idleResourceCh: make(chan idleResource[T], maxIdleSize), // 紀錄還回去幾個
		acquiredSizeCh: make(chan bool, maxIdleSize),            // 紀錄被拿幾個
		locker:         sync.Mutex{},
	}

	go rp.sweepIdleResource()
	return rp
}

func (rp *ResourcePool[T]) Acquire(ctx context.Context) (resource T, err error) {
	rp.locker.Lock()
	/*
		defer func() {
			if any(resource) != nil {
				go rp.listenCtxDone(ctx, resource)
			}

			rp.locker.Unlock()
		}()
	*/
	defer rp.locker.Unlock()

	select {
	case idle := <-rp.idleResourceCh: // 從閒置資源拿
		rp.acquiredSizeCh <- true
		resource = idle.value
		return
	default:
	}

	timeout := time.Now().Add(3 * time.Second)

Loop:
	for {
		select {
		case rp.acquiredSizeCh <- true:
			if resource, err = rp.creator(ctx); err != nil {
				<-rp.acquiredSizeCh
			}
			return
		default: // 使用中資源數量大於限制
			if time.Now().After(timeout) {
				err = errors.New("acquisition timeout error")
				return
			}
			goto Loop
		}
	}
}

func (rp *ResourcePool[T]) Release(resource T) {
	rp.locker.Lock()
	defer rp.locker.Unlock()

	rp.idleResourceCh <- idleResource[T]{
		sweepTime: time.Now().Add(rp.maxIdleTime),
		value:     resource,
	}

	<-rp.acquiredSizeCh
}

func (rp *ResourcePool[T]) NumIdle() int {
	rp.locker.Lock()
	defer rp.locker.Unlock()
	return len(rp.idleResourceCh)
}

func (rp *ResourcePool[T]) sweepIdleResource() {
	for {
		// 避免 Acquire() 拿不到閒置資源
		rp.locker.Lock()

		select {
		case resource := <-rp.idleResourceCh:

			// 檢查資源過期
			if time.Now().Before(resource.sweepTime) {
				rp.idleResourceCh <- resource
			} else {
				fmt.Println("sweep", &resource)
			}
		default:
		}

		rp.locker.Unlock()
	}
}

/*
func (rp *ResourcePool[T]) listenCtxDone(ctx context.Context, resource T) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("release resource", &resource)
			rp.Release(resource)
			return
		default:
		}
	}
}
*/
