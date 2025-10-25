package pkg1

import (
	"log"
	"os"
)

type MyType struct{}

func (m MyType) panic() {}

func errCheckFunc() {
	// формулируем ожидания: анализатор должен находить ошибку,
	// описанную в комментарии want
	panic("test") // want "usage of panic function is forbidden"

	if true {
		panic("test") // want "usage of panic function is forbidden"
	}

	m := MyType{}
	m.panic() // must be ignored

	os.Exit(1)        // want "os.Exit outside main"
	log.Fatal("test") // want "log.Fatal outside main"
}
