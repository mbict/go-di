package di

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

func New(init ...func(container *Container) error) (*Container, error) {
	c := &Container{
		factories: make(map[reflect.Type]func() error),
		instances: make(map[reflect.Type]reflect.Value),
		loading:   make(map[reflect.Type]bool),
	}

	for _, f := range init {
		if err := f(c); err != nil {
			return c, err
		}
	}
	return c, nil
}

type Finalizer func()

type Container struct {
	factories  map[reflect.Type]func() error
	instances  map[reflect.Type]reflect.Value
	finalizers []Finalizer
	loading    map[reflect.Type]bool
	mu         sync.Mutex
}

func (c *Container) load(t reflect.Type) (reflect.Value, error) {
	instance, found := c.instances[t]
	if found {
		return instance, nil
	}

	factory, found := c.factories[t]
	if !found {
		return reflect.Value{}, errors.New("could not find a factory for type " + t.String())
	}

	c.loading[t] = true
	defer func() { c.loading[t] = false }()

	err := factory()
	if err != nil {
		return reflect.Value{}, err
	}

	instance, found = c.instances[t]
	if !found {
		return reflect.Value{}, errors.New("factory did not produce a value for type " + t.String())
	}

	return instance, nil
}

func (c *Container) Get[T any]() T {
	instance, err := c.get[T]()
	if err != nil {
		panic(err)
	}
	return instance
}

func (c *Container) GetE[T any]() (T, error) {
	res, err := c.get[T]()
	if err != nil {
		return res, fmt.Errorf("could not get type `%s`, %w\n%s", reflect.TypeFor[T]().String(), err, stackTrace(1))
	}
	return res, nil
}

func (c *Container) get[T any]() (T, error) {
	t := reflect.TypeFor[T]()
	instance, err := c.load(t)
	if err != nil {
		var r T
		return r, err
	}
	return instance.Interface().(T), nil
}

// possible factory signatures
// func() (type)
// func() (type, Finalizer)
// func() (type1, type2, type3...)
// func() (type1, type2, type3..., Finalizer)
// func() (type, error)
// func() (type, Finalizer, error)
// func() (type1, type2, type3...., Finalizer, error)
// func(arg1) (type)
// func(arg1) (type, Finalizer)
// func(arg1, arg2, arg3....) (type)
// func(arg1, arg2, arg3....) (type, Finalizer)
// func(arg1) (type, error)
// func(arg1) (type, Finalizer, error)
// func(arg1, arg2, arg3....) (type, error)
// func(arg1, arg2, arg3....) (type, Finalizer, error)
// func(arg1) (type1, type2, type3,...., error)
// func(arg1) (type1, type2, type3,...., Finalizer, error)
// func(arg1, arg2, arg3....) (type1, type2, type3,... error)
// func(arg1, arg2, arg3....) (type1, type2, type3,..., Finalizer, error)
func (c *Container) Provides(factory any) error {
	factoryVal := reflect.ValueOf(factory)
	factoryType := factoryVal.Type()

	// Check if provided argument is a function
	if factoryType.Kind() != reflect.Func {
		return fmt.Errorf("provided argument is not a function")
	}

	// Determine if last return value is an error
	numOut := factoryType.NumOut()
	if numOut == 0 {
		return fmt.Errorf("factory must return at least one value")
	}

	lastReturnType := factoryType.Out(numOut - 1)
	isLastError := lastReturnType.Implements(reflect.TypeOf((*error)(nil)).Elem())

	// Check for Finalizer in return values
	finalizerType := reflect.TypeOf((*Finalizer)(nil)).Elem()
	hasFinalizer := false
	finalizerIdx := -1

	if isLastError {
		// Finalizer is second-to-last (before error)
		if numOut >= 2 {
			finalizerIdx = numOut - 2
			hasFinalizer = factoryType.Out(finalizerIdx) == finalizerType
		}
	} else {
		// Finalizer is last (no error)
		if numOut >= 2 {
			finalizerIdx = numOut - 1
			hasFinalizer = factoryType.Out(finalizerIdx) == finalizerType
		}
	}

	// Determine which return types to provides (exclude error and Finalizer)
	returnTypes := make([]reflect.Type, 0)
	excludeIdx := -1
	if isLastError {
		excludeIdx = numOut - 1
	}
	if hasFinalizer {
		excludeIdx = finalizerIdx
	}
	for i := 0; i < numOut; i++ {
		if i == excludeIdx {
			continue
		}
		returnTypes = append(returnTypes, factoryType.Out(i))
	}

	if len(returnTypes) == 0 {
		return fmt.Errorf("factory must return at least one non-error value")
	}

	// Create a callback function that will be used as the factory
	callback := func() error {
		// Resolve all arguments
		numArgs := factoryType.NumIn()
		args := make([]reflect.Value, numArgs)

		for i := 0; i < numArgs; i++ {
			argType := factoryType.In(i)

			// Check for circular dependency
			if c.loading[argType] {
				return fmt.Errorf("circular dependency detected: type %s is already being loaded", argType.String())
			}

			// Load the dependency
			argVal, err := c.load(argType)
			if err != nil {

				fmt.Println("could not load", argType.String())

				return err
			}

			args[i] = argVal
		}

		// Call the factory function
		results := factoryVal.Call(args)

		// Check if last result is an error
		if isLastError && len(results) > len(returnTypes) {
			lastResult := results[len(results)-1]
			if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
				if !lastResult.IsNil() {
					return fmt.Errorf("factory with signature `%s` returned error: %w", factoryVal.Type().String(), lastResult.Interface().(error))
				}
			}
		}

		// Process results
		resultIdx := 0
		for _, returnType := range returnTypes {
			if resultIdx >= len(results) {
				return fmt.Errorf("factory did not return expected number of values")
			}

			resultVal := results[resultIdx]
			c.instances[returnType] = resultVal
			resultIdx++
		}

		// Extract Finalizer if present and factory succeeded
		if hasFinalizer && finalizerIdx >= 0 {
			finalizerVal := results[finalizerIdx].Interface().(Finalizer)
			if finalizerVal != nil {
				c.mu.Lock()
				c.finalizers = append(c.finalizers, finalizerVal)
				c.mu.Unlock()
			}
		}

		return nil
	}

	// Provides the callback for each return type
	for _, returnType := range returnTypes {
		c.factories[returnType] = callback
	}

	return nil
}

func (c *Container) Invoke(fn any) {
	if err := c.invoke(fn); err != nil {
		panic(fmt.Errorf("%w\n%s", err, stackTrace(1)))
	}
}

func (c *Container) InvokeE(fn any) error {
	if err := c.invoke(fn); err != nil {
		return fmt.Errorf("%w\n%s", err, stackTrace(1))
	}
	return nil
}

func (c *Container) invoke(fn any) error {
	fnVal := reflect.ValueOf(fn)
	fnType := fnVal.Type()

	// Check if fn is a function
	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("provided argument is not a function")
	}

	// Get all argument types and create dummy values
	numArgs := fnType.NumIn()
	args := make([]reflect.Value, numArgs)

	// Load the arguments
	for i := 0; i < numArgs; i++ {
		argType := fnType.In(i)

		argInstance, err := c.load(argType)
		if err != nil {
			return errors.Join(fmt.Errorf("could not invoke `%T`", fn), err)
		}

		args[i] = argInstance
	}

	// Call the function with arguments
	results := fnVal.Call(args)

	// Check if function returns an error
	if fnType.NumOut() > 0 {
		lastResult := results[fnType.NumOut()-1]
		if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !lastResult.IsNil() {
				return lastResult.Interface().(error)
			}
		}
	}
	return nil
}

// Make calls the provided function with auto-resolved dependencies and returns its result typed as T.
// The function must have a compatible signature, optionally supporting error as the last return value.
// Supported fn signatures:
// func() T
// func(dep1) T
// func(dep1, dep2, dep3...) T
// func() (T, error)
// func(dep1) (T, error)
// func(dep1, dep2, dep3...) (T, error)
func (c *Container) Make[T any](fn any) T {
	res, err := c.make[T](fn)
	if err != nil {
		panic(fmt.Errorf("%w\n%s", err, stackTrace(1)))
	}
	return res
}

// MakeE calls fn with auto-resolved dependencies and returns its first return value typed as T.
// The function must have a compatible signature, optionally supporting error as the last return value.
// Supported fn signatures:
// func() T
// func(dep1) T
// func(dep1, dep2, dep3...) T
// func() (T, error)
// func(dep1) (T, error)
// func(dep1, dep2, dep3...) (T, error)
func (c *Container) MakeE[T any](fn any) (T, error) {
	res, err := c.make[T](fn)
	if err != nil {
		return res, fmt.Errorf("%w\n%s", err, stackTrace(1))
	}
	return res, nil
}

func (c *Container) make[T any](fn any) (T, error) {
	fnVal := reflect.ValueOf(fn)
	fnType := fnVal.Type()

	var zero T

	// Check if fn is a function
	if fnType.Kind() != reflect.Func {
		return zero, fmt.Errorf("provided argument is not a function")
	}

	// fn must return at least one value
	numOut := fnType.NumOut()
	if numOut == 0 {
		return zero, fmt.Errorf("fn must return at least one value")
	}

	// Load the arguments
	numArgs := fnType.NumIn()
	args := make([]reflect.Value, numArgs)

	for i := 0; i < numArgs; i++ {
		argType := fnType.In(i)

		argInstance, err := c.load(argType)
		if err != nil {
			return zero, errors.Join(fmt.Errorf("could not call result fn `%T`", fn), err)
		}

		args[i] = argInstance
	}

	// Call the function with arguments
	results := fnVal.Call(args)

	// Check if last return value is an error
	lastResult := results[numOut-1]
	if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		if !lastResult.IsNil() {
			return zero, lastResult.Interface().(error)
		}
	}

	// Return first result typed as T
	return results[0].Interface().(T), nil
}

func (c *Container) Finalize() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, f := range c.finalizers {
		f()
	}
	c.finalizers = nil
	for k := range c.instances {
		delete(c.instances, k)
	}
}
