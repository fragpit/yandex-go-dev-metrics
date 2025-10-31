package pool_test

import (
	"fmt"

	"github.com/fragpit/yandex-go-dev-metrics/pkg/pool"
)

type Buffer struct {
	data []byte
}

func (b *Buffer) Reset() {
	b.data = b.data[:0]
}

func (b *Buffer) Write(p []byte) {
	b.data = append(b.data, p...)
}

func (b *Buffer) String() string {
	return string(b.data)
}

func ExamplePool() {
	// [] type added for clarity, as far as I see, compiler can get the type
	// automatically.
	bufferPool := pool.New[*Buffer](func() *Buffer {
		return &Buffer{
			data: make([]byte, 0, 64),
		}
	})

	buf := bufferPool.Get()
	buf.Write([]byte("Hello, World!"))
	fmt.Println(buf.String())

	bufferPool.Put(buf)

	buf2 := bufferPool.Get()
	fmt.Println(len(buf2.data))

	// Output:
	// Hello, World!
	// 0
}
