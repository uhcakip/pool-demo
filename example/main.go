package main

import (
	"context"
	"dcard-resource-pool/pool"
	"fmt"
	"time"
)

func main() {
	type item struct {
		name string
		code string
	}

	ctx := context.Background()
	idleSize := 1
	idleTime := 1000 * time.Millisecond
	creator := func(ctx context.Context) (any, error) {
		return item{}, nil
	}

	p := pool.New(creator, idleSize, idleTime)
	obj, err := p.Acquire(ctx)

	if err != nil {
		panic(err)
	}

	p.Release(obj)
	fmt.Println(p.NumIdle()) // 1
	time.Sleep(idleTime)
	fmt.Println(p.NumIdle()) // 0
}
