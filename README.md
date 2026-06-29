# di

A small constructor-based dependency injection container for Go.

It helps you register factories, resolve dependencies by type, preconfigure a container during construction, invoke functions with injected arguments, and run optional cleanup hooks (`Finalizer`) during shutdown.

## Go Version Requirement

> This library is supported only on **Go 1.27 and up**.

The API uses generic struct methods (for example `Get[T]()` and `GetE[T]()`), so older Go versions are not supported.

## Use Cases

- Wiring app components without manual object graph construction.
- Keeping service setup explicit via constructors.
- Running startup routines with auto-injected dependencies.
- Registering cleanup logic for resources (DB clients, queues, etc.).

## Installation

This repository module is `github.com/mbict/go-di`, so import directly:

```go
import "github.com/mbict/go-di"
```

If you publish this package as a separate module later, use that module path with `go get` and update the import accordingly.

## Usage Example

```go
package main

import (
	"fmt"
	"log"

	"github.com/mbict/go-di"
)

type Config struct {
	DSN string
}

type DB struct {
	dsn string
}

type Service struct {
	db *DB
}

func main() {
	c, err := di.New(
		func(container *di.Container) error {
			return container.Provides(di.Instance(Config{DSN: "postgres://localhost/app"}))
		},
		func(container *di.Container) error {
			return container.Provides(func(cfg Config) (*DB, di.Finalizer, error) {
				db := &DB{dsn: cfg.DSN}
				finalizer := func() {
					fmt.Println("closing db:", db.dsn)
				}
				return db, finalizer, nil
			})
		},
		func(container *di.Container) error {
			return container.Provides(func(db *DB) *Service {
				return &Service{db: db}
			})
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	// Run registered finalizers on exit
    defer c.Finalize()
	    	
	// Resolve by type.
	svc, err := c.GetE[*Service]()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("service ready with db:", svc.db.dsn)

	// Invoke function with injected deps.
	if err := c.InvokeE(func(s *Service) error {
		fmt.Println("invoked with service:", s.db.dsn)
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	// Call a function with injected deps and return its result directly.
	dsn, err := c.ResultE[string](func(db *DB) (string, error) {
		return db.dsn, nil
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("resolved DSN via ResultE:", dsn)

	// Same without error handling — panics on failure.
	dsn2 := c.Result[string](func(db *DB) string {
		return db.dsn
	})
	fmt.Println("resolved DSN via Result:", dsn2)
}
```

## Core API

- `New(init ...func(container *Container) error) (*Container, error)`
- `Provides(factory any) error`
- `Get[T any]() T`
- `GetE[T any]() (T, error)`
- `Invoke(fn any)`
- `InvokeE(fn any) error`
- `Result[T any](fn any) T`
- `ResultE[T any](fn any) (T, error)`
- `Instance[T any](instance T) func() T`
- `Alias[T any, A any]() func(T) A`
- `Finalize()`

## Method Overview

| Method | Description |
|---|---|
| `New` | Create a container and optionally run init callbacks in order. Returns the container plus the first init error, if any. |
| `Provides` | Register a factory function that produces one or more types. |
| `Get[T]` | Resolve `T` from the container. Panics on error. |
| `GetE[T]` | Resolve `T` from the container. Returns error with stack trace. |
| `Invoke` | Call `fn` with auto-resolved arguments. Panics on error. |
| `InvokeE` | Call `fn` with auto-resolved arguments. Returns error with stack trace. |
| `Result[T]` | Call `fn` with auto-resolved arguments and return its first result as `T`. Panics on error. |
| `ResultE[T]` | Call `fn` with auto-resolved arguments and return its first result as `T`. Returns error with stack trace. |
| `Instance[T]` | Helper that turns an existing value into a zero-arg provider function. |
| `Alias[T, A]` | Helper that exposes a provided concrete type `T` as another type `A` such as an interface. |
| `Finalize` | Run all registered `Finalizer` callbacks and clear resolved instances. |

## Supported `Provides` Signatures

`Provides` accepts factory functions with zero or more dependencies (`arg1`, `arg2`, ...),
one or more produced values (`type`, `type1`, ...), and optional trailing `Finalizer`
and/or `error` return values.

Supported forms:

```
func() type
func() (type, Finalizer)
func() (type1, type2, type3...)
func() (type1, type2, type3..., Finalizer)
func() (type, error)
func() (type, Finalizer, error)
func() (type1, type2, type3..., Finalizer, error)

func(arg1) type
func(arg1) (type, Finalizer)
func(arg1, arg2, arg3...) type
func(arg1, arg2, arg3...) (type, Finalizer)
func(arg1) (type, error)
func(arg1) (type, Finalizer, error)
func(arg1, arg2, arg3...) (type, error)
func(arg1, arg2, arg3...) (type, Finalizer, error)

func(arg1) (type1, type2, type3..., error)
func(arg1) (type1, type2, type3..., Finalizer, error)
func(arg1, arg2, arg3...) (type1, type2, type3..., error)
func(arg1, arg2, arg3...) (type1, type2, type3..., Finalizer, error)
```

Notes:

- `error`, when present, must be the last return value.
- `Finalizer`, when present with `error`, appears before `error`.
- A factory must produce at least one non-`error` value.

## New(init ...)

`New` accepts any number of init callbacks:

```go
func(container *di.Container) error
```

Each init callback runs in order and can pre-register providers on the freshly created container.
If one callback returns an error, `New` stops immediately and returns:

- the partially initialized container
- the error from the failing init callback

This is useful for bootstrapping application wiring in one place.

```go
c, err := di.New(
	func(container *di.Container) error {
		return container.Provides(di.Instance("postgres://localhost/app"))
	},
	func(container *di.Container) error {
		return container.Provides(func(dsn string) *DB {
			return &DB{dsn: dsn}
		})
	},
)
if err != nil {
	log.Fatal(err)
}
```

## Helper Functions

### Instance

`Instance` wraps an already-created value and turns it into a provider function.

```go
c, err := di.New(
	func(container *di.Container) error {
		return container.Provides(di.Instance(Config{DSN: "postgres://localhost/app"}))
	},
)
if err != nil {
	log.Fatal(err)
}
```

This is especially useful for static config values, singletons you already created, or values loaded outside the container.

### Alias

`Alias` lets you expose one provided type as another type, typically a concrete type as an interface.

```go
type Greeter interface {
	Greet() string
}

type Service struct{}

func (Service) Greet() string { return "hello" }

c, err := di.New(
	func(container *di.Container) error {
		return container.Provides(di.Instance(Service{}))
	},
	func(container *di.Container) error {
		return container.Provides(di.Alias[Service, Greeter]())
	},
)
if err != nil {
	log.Fatal(err)
}

greeter, err := c.GetE[Greeter]()
if err != nil {
	log.Fatal(err)
}

fmt.Println(greeter.Greet())
```

## Result / ResultE

`Result` and `ResultE` are like `Invoke` / `InvokeE` but they return a typed value from `fn`.
Use them when you need the output of a function that takes injected dependencies.

Supported `fn` signatures:

```
func() T
func(dep1) T
func(dep1, dep2, dep3...) T
func() (T, error)
func(dep1) (T, error)
func(dep1, dep2, dep3...) (T, error)
```

## Error Behavior

- `GetE`, `InvokeE`, and `ResultE` return wrapped errors with an embedded stack trace pointing to the callsite.
- `Get`, `Invoke`, and `Result` panic on errors.
- Circular dependencies are detected during argument resolution.

