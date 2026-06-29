package di

// Instance returns a closure that always provides the same instance of the given generic type T.
func Instance[T any](instance T) func() T {
	return func() T {
		return instance
	}
}

// Alias allows you to create an alias for a type. This is useful when you want to provide the same instance for multiple types, or when you want to provide an instance for an interface that is implemented by a struct.
func Alias[T any, A any]() func(T) A {
	return func(t T) A {
		return any(t).(A)
	}
}
