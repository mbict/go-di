package di

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testA interface {
	Test() string
}

type testB struct {
	val string
}

func (t *testB) Test() string {
	return t.val
}

type testC func()
type testD func()

func x() testC {
	return func() {
		fmt.Println("testC")
	}
}
func y() testD {
	return func() {
		fmt.Println("testD")
	}
}

func Test_container_Getx(t *testing.T) {
	c := NewContainer(func(builder ContainerBuilder) {
		builder.Provide(x)
		builder.Provide(y)
	})

	x1 := Get[testC](c)
	x1()

	y1 := Get[testD](c)
	y1()

	x1 = Get[testC](c)
	x1()

	y1 = Get[testD](c)
	y1()
}

func Test_container_Get(t *testing.T) {
	mustProvide := &testB{}

	c := NewContainer(func(builder ContainerBuilder) {
		builder.Provide(func() testA {
			return mustProvide
		})
	})

	x := Get[testA](c)

	assert.Same(t, mustProvide, x)
}
