package pool

import "sync"

type Resettable interface {
	Reset()
}

type Pool[T Resettable] struct {
	pool sync.Pool
}

func New[T Resettable](newFunc func() T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return newFunc()
			},
		},
	}
}

func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

func (p *Pool[T]) Put(obj T) {
	obj.Reset()
	p.pool.Put(obj)
}
