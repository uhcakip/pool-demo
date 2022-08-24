package pool

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type testItem struct {
	Name string
	Code string
}

var creator = func(ctx context.Context) (any, error) {
	return testItem{}, nil
}

const (
	testSize = 5
	testTime = time.Second
)

func newPool() *ResourcePool[any] {
	pool := New(creator, testSize, testTime)
	return pool.(*ResourcePool[any])
}

func TestNew(t *testing.T) {
	pool := New(creator, 0, 0).(*ResourcePool[any])

	assert.Equal(t, defaultMaxIdleSize, pool.maxIdleSize)
	assert.Equal(t, defaultMaxIdleTime, pool.maxIdleTime)
}

func TestAcquire(t *testing.T) {
	pool := newPool()
	item, _ := pool.Acquire(context.Background())

	assert.IsType(t, testItem{}, item)
}

func TestAcquireTimeout(t *testing.T) {
	var err error
	ctx := context.Background()
	pool := newPool()

	for i := 0; i < testSize+1; i++ {
		if _, err = pool.Acquire(ctx); err != nil {
			break
		}
	}

	assert.Equal(t, ErrAcquistionTimeout, err)
}

func TestAcquireIdleResource(t *testing.T) {
	var item any
	ctx := context.Background()
	pool := newPool()

	for i := 0; i < testSize; i++ {
		item, _ = pool.Acquire(ctx)
	}

	pool.Release(item)
	item, _ = pool.Acquire(ctx)

	assert.Equal(t, 0, pool.NumIdle())
}

func TestAcquireCreatorError(t *testing.T) {
	creatorErr := func(ctx context.Context) (any, error) {
		return nil, errors.New("")
	}

	pool := New(creatorErr, testSize, testTime).(*ResourcePool[any])
	_, err := pool.Acquire(context.Background())
	assert.Error(t, err)
	assert.Equal(t, 0, len(pool.acquiredSizeCh))
}

func TestSweepIdleResource(t *testing.T) {
	pool := newPool()
	item, _ := pool.Acquire(context.Background())
	pool.Release(item)
	time.Sleep(testTime)

	assert.Equal(t, 0, pool.NumIdle())
}
