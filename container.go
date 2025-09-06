package di

import (
	"errors"
	"reflect"

	"github.com/mbict/go-requestbus"

	"go.uber.org/dig"
)

var DefaultContainer Container

type Container interface {
	Invoke(function interface{})
	InvokeE(function interface{}) error
}

type ContainerBuilder interface {
	Container
	Provide(constructor interface{}, opts ...dig.ProvideOption) error

	RegisterHandler(handler any) error
}

// Get retrieves a typed instance from the container.
func Get[T any](container ...Container) (val T) {
	if len(container) == 0 {
		container = []Container{DefaultContainer}
	}

	var err error
	for _, c := range container {
		err = c.InvokeE(func(a T) {
			val = a
		})

		if err == nil {
			return val
		}
	}

	if err != nil {
		panic(err)
	}

	return
}

// Instance wraps an object instance with a factory function.
func Instance[T any](instance T) func() T {
	return func() T {
		return instance
	}
}

func Alias[T any, A any]() func(T) A {
	return func(t T) A {
		return any(t).(A)
	}
}

type container struct {
	*dig.Container
}

func (c *container) Invoke(function interface{}) {
	must(c.Container.Invoke(function))
}

func (c *container) InvokeE(function interface{}) error {
	return c.Container.Invoke(function)
}

func (c *container) RegisterHandler(handler any) error {

	//check if the handler is a factory
	ht := reflect.TypeOf(handler)
	if ht.NumOut() != 1 {
		return errors.New("incompatible handler factory signature")
	}

	//add the handler to the container
	if err := c.Provide(handler); err != nil {
		return err
	}

	//make the constructor function to register the handler to the requestBus
	ft := reflect.FuncOf([]reflect.Type{
		ht.Out(0),
		reflect.TypeFor[*requestbus.RequestBus](),
		//reflect.TypeOf((*requestbus.RequestBus)(nil)).Elem(),
	}, []reflect.Type{
		reflect.TypeFor[error](),
	}, false)

	fn := reflect.MakeFunc(ft, func(args []reflect.Value) []reflect.Value {
		requestHandler := args[0].Interface()
		bus := args[1].Interface().(*requestbus.RequestBus)

		err := bus.RegisterHandler(requestHandler)
		if err != nil {
			return []reflect.Value{
				reflect.ValueOf(err),
			}
		}
		return []reflect.Value{
			reflect.Zero(reflect.TypeFor[error]()),
		}
	})

	return c.InvokeE(fn.Interface())
}

func NewContainer(init func(builder ContainerBuilder)) Container {
	c := &container{
		Container: dig.New(),
	}

	init(c)

	return c
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
