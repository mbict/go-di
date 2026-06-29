package di

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type dep1 struct{ A int }
type dep2 struct{ B string }
type dep3 struct{ C bool }

type aliasGreeter interface {
	Greet() string
}

type aliasGreeterImpl struct {
	message string
}

func (g aliasGreeterImpl) Greet() string {
	return g.message
}

func providesDeps(t *testing.T, c *Container) {
	t.Helper()
	assert.NoError(t, c.Provides(func() dep1 { return dep1{A: 7} }))
	assert.NoError(t, c.Provides(func() dep2 { return dep2{B: "dep"} }))
	assert.NoError(t, c.Provides(func() dep3 { return dep3{C: true} }))
}

func mustGetE[T any](t *testing.T, c *Container) T {
	t.Helper()
	v, err := c.GetE[T]()
	assert.NoError(t, err)
	return v
}

func mustNew(t *testing.T, init ...func(container *Container) error) *Container {
	t.Helper()
	c, err := New(init...)
	assert.NoError(t, err)
	assert.NotNil(t, c)
	return c
}

func TestProvides_SupportedSignatures(t *testing.T) {
	t.Run("func() (type)", func(t *testing.T) {
		type out struct{ V string }
		c := mustNew(t)
		assert.NoError(t, c.Provides(func() out { return out{V: "ok"} }))
		assert.Equal(t, out{V: "ok"}, mustGetE[out](t, c))
	})

	t.Run("func() (type, Finalizer)", func(t *testing.T) {
		type out struct{ V string }
		c := mustNew(t)
		finalized := false
		assert.NoError(t, c.Provides(func() (out, Finalizer) {
			return out{V: "ok"}, func() { finalized = true }
		}))
		assert.Equal(t, out{V: "ok"}, mustGetE[out](t, c))
		c.Finalize()
		assert.True(t, finalized)
	})

	t.Run("func() (type1, type2, type3...)", func(t *testing.T) {
		type out1 struct{ V int }
		type out2 struct{ V string }
		type out3 struct{ V bool }
		c := mustNew(t)
		assert.NoError(t, c.Provides(func() (out1, out2, out3) {
			return out1{V: 1}, out2{V: "two"}, out3{V: true}
		}))
		assert.Equal(t, out1{V: 1}, mustGetE[out1](t, c))
		assert.Equal(t, out2{V: "two"}, mustGetE[out2](t, c))
		assert.Equal(t, out3{V: true}, mustGetE[out3](t, c))
	})

	t.Run("func() (type1, type2, type3..., Finalizer)", func(t *testing.T) {
		type out1 struct{ V int }
		type out2 struct{ V string }
		type out3 struct{ V bool }
		c := mustNew(t)
		finalized := false
		assert.NoError(t, c.Provides(func() (out1, out2, out3, Finalizer) {
			return out1{V: 1}, out2{V: "two"}, out3{V: true}, func() { finalized = true }
		}))
		assert.Equal(t, out1{V: 1}, mustGetE[out1](t, c))
		assert.Equal(t, out2{V: "two"}, mustGetE[out2](t, c))
		assert.Equal(t, out3{V: true}, mustGetE[out3](t, c))
		c.Finalize()
		assert.True(t, finalized)
	})

	t.Run("func() (type, error)", func(t *testing.T) {
		type out struct{ V string }
		c := mustNew(t)
		assert.NoError(t, c.Provides(func() (out, error) {
			return out{V: "ok"}, nil
		}))
		assert.Equal(t, out{V: "ok"}, mustGetE[out](t, c))
	})

	t.Run("func() (type, Finalizer, error)", func(t *testing.T) {
		type out struct{ V string }
		c := mustNew(t)
		finalized := false
		assert.NoError(t, c.Provides(func() (out, Finalizer, error) {
			return out{V: "ok"}, func() { finalized = true }, nil
		}))
		assert.Equal(t, out{V: "ok"}, mustGetE[out](t, c))
		c.Finalize()
		assert.True(t, finalized)
	})

	t.Run("func() (type1, type2, type3...., Finalizer, error)", func(t *testing.T) {
		type out1 struct{ V int }
		type out2 struct{ V string }
		type out3 struct{ V bool }
		c := mustNew(t)
		finalized := false
		assert.NoError(t, c.Provides(func() (out1, out2, out3, Finalizer, error) {
			return out1{V: 1}, out2{V: "two"}, out3{V: true}, func() { finalized = true }, nil
		}))
		assert.Equal(t, out1{V: 1}, mustGetE[out1](t, c))
		assert.Equal(t, out2{V: "two"}, mustGetE[out2](t, c))
		assert.Equal(t, out3{V: true}, mustGetE[out3](t, c))
		c.Finalize()
		assert.True(t, finalized)
	})

	t.Run("func(arg1) (type)", func(t *testing.T) {
		type out struct{ V int }
		c := mustNew(t)
		providesDeps(t, c)
		assert.NoError(t, c.Provides(func(d dep1) out { return out{V: d.A} }))
		assert.Equal(t, out{V: 7}, mustGetE[out](t, c))
	})

	t.Run("func(arg1) (type, Finalizer)", func(t *testing.T) {
		type out struct{ V int }
		c := mustNew(t)
		providesDeps(t, c)
		finalized := false
		assert.NoError(t, c.Provides(func(d dep1) (out, Finalizer) {
			return out{V: d.A}, func() { finalized = true }
		}))
		assert.Equal(t, out{V: 7}, mustGetE[out](t, c))
		c.Finalize()
		assert.True(t, finalized)
	})

	t.Run("func(arg1, arg2, arg3....) (type)", func(t *testing.T) {
		type out struct{ V string }
		c := mustNew(t)
		providesDeps(t, c)
		assert.NoError(t, c.Provides(func(a dep1, b dep2, d dep3) out {
			return out{V: b.B}
		}))
		assert.Equal(t, out{V: "dep"}, mustGetE[out](t, c))
	})

	t.Run("func(arg1, arg2, arg3....) (type, Finalizer)", func(t *testing.T) {
		type out struct{ V bool }
		c := mustNew(t)
		providesDeps(t, c)
		finalized := false
		assert.NoError(t, c.Provides(func(a dep1, b dep2, d dep3) (out, Finalizer) {
			return out{V: d.C}, func() { finalized = true }
		}))
		assert.Equal(t, out{V: true}, mustGetE[out](t, c))
		c.Finalize()
		assert.True(t, finalized)
	})

	t.Run("func(arg1) (type, error)", func(t *testing.T) {
		type out struct{ V int }
		c := mustNew(t)
		providesDeps(t, c)
		assert.NoError(t, c.Provides(func(d dep1) (out, error) {
			return out{V: d.A}, nil
		}))
		assert.Equal(t, out{V: 7}, mustGetE[out](t, c))
	})

	t.Run("func(arg1) (type, Finalizer, error)", func(t *testing.T) {
		type out struct{ V int }
		c := mustNew(t)
		providesDeps(t, c)
		finalized := false
		assert.NoError(t, c.Provides(func(d dep1) (out, Finalizer, error) {
			return out{V: d.A}, func() { finalized = true }, nil
		}))
		assert.Equal(t, out{V: 7}, mustGetE[out](t, c))
		c.Finalize()
		assert.True(t, finalized)
	})

	t.Run("func(arg1, arg2, arg3....) (type, error)", func(t *testing.T) {
		type out struct{ V string }
		c := mustNew(t)
		providesDeps(t, c)
		assert.NoError(t, c.Provides(func(a dep1, b dep2, d dep3) (out, error) {
			return out{V: b.B}, nil
		}))
		assert.Equal(t, out{V: "dep"}, mustGetE[out](t, c))
	})

	t.Run("func(arg1, arg2, arg3....) (type, Finalizer, error)", func(t *testing.T) {
		type out struct{ V bool }
		c := mustNew(t)
		providesDeps(t, c)
		finalized := false
		assert.NoError(t, c.Provides(func(a dep1, b dep2, d dep3) (out, Finalizer, error) {
			return out{V: d.C}, func() { finalized = true }, nil
		}))
		assert.Equal(t, out{V: true}, mustGetE[out](t, c))
		c.Finalize()
		assert.True(t, finalized)
	})

	t.Run("func(arg1) (type1, type2, type3,...., error)", func(t *testing.T) {
		type out1 struct{ V int }
		type out2 struct{ V string }
		type out3 struct{ V bool }
		c := mustNew(t)
		providesDeps(t, c)
		assert.NoError(t, c.Provides(func(a dep1) (out1, out2, out3, error) {
			return out1{V: a.A}, out2{V: "two"}, out3{V: true}, nil
		}))
		assert.Equal(t, out1{V: 7}, mustGetE[out1](t, c))
		assert.Equal(t, out2{V: "two"}, mustGetE[out2](t, c))
		assert.Equal(t, out3{V: true}, mustGetE[out3](t, c))
	})

	t.Run("func(arg1) (type1, type2, type3,...., Finalizer, error)", func(t *testing.T) {
		type out1 struct{ V int }
		type out2 struct{ V string }
		type out3 struct{ V bool }
		c := mustNew(t)
		providesDeps(t, c)
		finalized := false
		assert.NoError(t, c.Provides(func(a dep1) (out1, out2, out3, Finalizer, error) {
			return out1{V: a.A}, out2{V: "two"}, out3{V: true}, func() { finalized = true }, nil
		}))
		assert.Equal(t, out1{V: 7}, mustGetE[out1](t, c))
		assert.Equal(t, out2{V: "two"}, mustGetE[out2](t, c))
		assert.Equal(t, out3{V: true}, mustGetE[out3](t, c))
		c.Finalize()
		assert.True(t, finalized)
	})

	t.Run("func(arg1, arg2, arg3....) (type1, type2, type3,... error)", func(t *testing.T) {
		type out1 struct{ V int }
		type out2 struct{ V string }
		type out3 struct{ V bool }
		c := mustNew(t)
		providesDeps(t, c)
		assert.NoError(t, c.Provides(func(a dep1, b dep2, d dep3) (out1, out2, out3, error) {
			return out1{V: a.A}, out2{V: b.B}, out3{V: d.C}, nil
		}))
		assert.Equal(t, out1{V: 7}, mustGetE[out1](t, c))
		assert.Equal(t, out2{V: "dep"}, mustGetE[out2](t, c))
		assert.Equal(t, out3{V: true}, mustGetE[out3](t, c))
	})

	t.Run("func(arg1, arg2, arg3....) (type1, type2, type3,..., Finalizer, error)", func(t *testing.T) {
		type out1 struct{ V int }
		type out2 struct{ V string }
		type out3 struct{ V bool }
		c := mustNew(t)
		providesDeps(t, c)
		finalized := false
		assert.NoError(t, c.Provides(func(a dep1, b dep2, d dep3) (out1, out2, out3, Finalizer, error) {
			return out1{V: a.A}, out2{V: b.B}, out3{V: d.C}, func() { finalized = true }, nil
		}))
		assert.Equal(t, out1{V: 7}, mustGetE[out1](t, c))
		assert.Equal(t, out2{V: "dep"}, mustGetE[out2](t, c))
		assert.Equal(t, out3{V: true}, mustGetE[out3](t, c))
		c.Finalize()
		assert.True(t, finalized)
	})
}

func TestProvides_UnsupportedSignatures(t *testing.T) {
	t.Run("rejects non-function", func(t *testing.T) {
		c := mustNew(t)
		err := c.Provides(123)
		assert.Error(t, err)
	})

	t.Run("rejects func() with no return values", func(t *testing.T) {
		c := mustNew(t)
		err := c.Provides(func() {})
		assert.Error(t, err)
	})

	t.Run("rejects func() error-only", func(t *testing.T) {
		c := mustNew(t)
		err := c.Provides(func() error { return nil })
		assert.Error(t, err)
	})

}

func TestGetE_MissingTypeIncludesStackTrace(t *testing.T) {
	type missing struct{}

	c := mustNew(t)
	_, err := c.GetE[missing]()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not get type `di.missing`")
	assert.Contains(t, err.Error(), "could not find a factory for type di.missing")
	assert.Contains(t, err.Error(), "TestGetE_MissingTypeIncludesStackTrace")
	assert.Contains(t, err.Error(), "container_test.go")
}

func TestGet_MissingTypePanics(t *testing.T) {
	type missing struct{}

	c := mustNew(t)
	assert.PanicsWithError(
		t,
		"could not find a factory for type di.missing",
		func() { _ = c.Get[missing]() },
	)
}

func TestInvokeE_RequestedSignatures_Success(t *testing.T) {
	t.Run("func()", func(t *testing.T) {
		c := mustNew(t)
		called := false

		err := c.InvokeE(func() {
			called = true
		})

		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("func() error", func(t *testing.T) {
		c := mustNew(t)
		called := false

		err := c.InvokeE(func() error {
			called = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("func(dep)", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)

		called := false
		got := dep1{}
		err := c.InvokeE(func(d dep1) {
			called = true
			got = d
		})

		assert.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, dep1{A: 7}, got)
	})

	t.Run("func(dep1, dep2)", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)

		called := false
		got1 := dep1{}
		got2 := dep2{}
		err := c.InvokeE(func(a dep1, b dep2) {
			called = true
			got1 = a
			got2 = b
		})

		assert.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, dep1{A: 7}, got1)
		assert.Equal(t, dep2{B: "dep"}, got2)
	})

	t.Run("func(dep) error", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)

		called := false
		got := dep1{}
		err := c.InvokeE(func(d dep1) error {
			called = true
			got = d
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, dep1{A: 7}, got)
	})

	t.Run("func(dep1, dep2) error", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)

		called := false
		got1 := dep1{}
		got2 := dep2{}
		err := c.InvokeE(func(a dep1, b dep2) error {
			called = true
			got1 = a
			got2 = b
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, dep1{A: 7}, got1)
		assert.Equal(t, dep2{B: "dep"}, got2)
	})

}

func TestInvokeE_RequestedSignatures_Failure(t *testing.T) {
	t.Run("func(dep) unresolved dependency", func(t *testing.T) {
		type missing struct{ X int }

		c := mustNew(t)
		called := false

		err := c.InvokeE(func(_ missing) {
			called = true
		})

		assert.Error(t, err)
		assert.False(t, called)
		assert.Contains(t, err.Error(), "could not invoke")
		assert.Contains(t, err.Error(), "could not find a factory for type di.missing")
	})

	t.Run("func(dep1, dep2) unresolved dependencies", func(t *testing.T) {
		type missing1 struct{ X int }
		type missing2 struct{ Y string }

		c := mustNew(t)
		called := false

		err := c.InvokeE(func(_ missing1, _ missing2) {
			called = true
		})

		assert.Error(t, err)
		assert.False(t, called)
		assert.Contains(t, err.Error(), "could not invoke")
		assert.Contains(t, err.Error(), "could not find a factory for type")
	})
}

func TestInvoke_RequestedSignatures_Success(t *testing.T) {
	t.Run("func()", func(t *testing.T) {
		c := mustNew(t)
		called := false

		assert.NotPanics(t, func() {
			c.Invoke(func() {
				called = true
			})
		})
		assert.True(t, called)
	})

	t.Run("func() error", func(t *testing.T) {
		c := mustNew(t)
		called := false

		assert.NotPanics(t, func() {
			c.Invoke(func() error {
				called = true
				return nil
			})
		})
		assert.True(t, called)
	})

	t.Run("func(dep)", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)

		called := false
		got := dep1{}
		assert.NotPanics(t, func() {
			c.Invoke(func(d dep1) {
				called = true
				got = d
			})
		})

		assert.True(t, called)
		assert.Equal(t, dep1{A: 7}, got)
	})

	t.Run("func(dep1, dep2)", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)

		called := false
		got1 := dep1{}
		got2 := dep2{}
		assert.NotPanics(t, func() {
			c.Invoke(func(a dep1, b dep2) {
				called = true
				got1 = a
				got2 = b
			})
		})

		assert.True(t, called)
		assert.Equal(t, dep1{A: 7}, got1)
		assert.Equal(t, dep2{B: "dep"}, got2)
	})

	t.Run("func(dep) error", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)

		called := false
		got := dep1{}
		assert.NotPanics(t, func() {
			c.Invoke(func(d dep1) error {
				called = true
				got = d
				return nil
			})
		})

		assert.True(t, called)
		assert.Equal(t, dep1{A: 7}, got)
	})

	t.Run("func(dep1, dep2) error", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)

		called := false
		got1 := dep1{}
		got2 := dep2{}
		assert.NotPanics(t, func() {
			c.Invoke(func(a dep1, b dep2) error {
				called = true
				got1 = a
				got2 = b
				return nil
			})
		})

		assert.True(t, called)
		assert.Equal(t, dep1{A: 7}, got1)
		assert.Equal(t, dep2{B: "dep"}, got2)
	})

}

func TestInvoke_RequestedSignatures_Failure(t *testing.T) {
	t.Run("func(dep) unresolved dependency panics", func(t *testing.T) {
		type missing struct{ X int }

		c := mustNew(t)
		called := false

		assert.Panics(t, func() {
			c.Invoke(func(_ missing) {
				called = true
			})
		})
		assert.False(t, called)
	})

	t.Run("func(dep1, dep2) unresolved dependencies panics", func(t *testing.T) {
		type missing1 struct{ X int }
		type missing2 struct{ Y string }

		c := mustNew(t)
		called := false

		assert.Panics(t, func() {
			c.Invoke(func(_ missing1, _ missing2) {
				called = true
			})
		})
		assert.False(t, called)
	})
}

func TestInvokeE_AdditionalSignatures(t *testing.T) {
	t.Run("func() (type)", func(t *testing.T) {
		c := mustNew(t)
		called := false

		err := c.InvokeE(func() int {
			called = true
			return 42
		})

		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("func(dep) (type)", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)

		called := false
		got := dep1{}
		err := c.InvokeE(func(d dep1) int {
			called = true
			got = d
			return d.A + 1
		})

		assert.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, dep1{A: 7}, got)
	})

	t.Run("func(dep1, dep2) (type1, type2)", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)

		called := false
		got1 := dep1{}
		got2 := dep2{}
		err := c.InvokeE(func(a dep1, b dep2) (int, string) {
			called = true
			got1 = a
			got2 = b
			return a.A, b.B
		})

		assert.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, dep1{A: 7}, got1)
		assert.Equal(t, dep2{B: "dep"}, got2)
	})

	t.Run("func() (type, error) nil", func(t *testing.T) {
		c := mustNew(t)
		called := false

		err := c.InvokeE(func() (int, error) {
			called = true
			return 7, nil
		})

		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("func() (type, error) non-nil", func(t *testing.T) {
		c := mustNew(t)
		called := false

		err := c.InvokeE(func() (int, error) {
			called = true
			return 0, errors.New("invokee failed")
		})

		assert.Error(t, err)
		assert.True(t, called)
		assert.Contains(t, err.Error(), "invokee failed")
	})

	t.Run("func(dep1, dep2) (type, error) nil", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)

		called := false
		got1 := dep1{}
		got2 := dep2{}
		err := c.InvokeE(func(a dep1, b dep2) (string, error) {
			called = true
			got1 = a
			got2 = b
			return b.B, nil
		})

		assert.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, dep1{A: 7}, got1)
		assert.Equal(t, dep2{B: "dep"}, got2)
	})

	t.Run("func(dep1, dep2) (type, error) non-nil", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)

		called := false
		got1 := dep1{}
		got2 := dep2{}
		err := c.InvokeE(func(a dep1, b dep2) (string, error) {
			called = true
			got1 = a
			got2 = b
			return "", errors.New("dep invokee failed")
		})

		assert.Error(t, err)
		assert.True(t, called)
		assert.Equal(t, dep1{A: 7}, got1)
		assert.Equal(t, dep2{B: "dep"}, got2)
		assert.Contains(t, err.Error(), "dep invokee failed")
	})
}

func TestInvoke_AdditionalSignatures(t *testing.T) {
	t.Run("func() (type)", func(t *testing.T) {
		c := mustNew(t)
		called := false

		assert.NotPanics(t, func() {
			c.Invoke(func() int {
				called = true
				return 42
			})
		})
		assert.True(t, called)
	})

	t.Run("func(dep) (type)", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)

		called := false
		got := dep1{}
		assert.NotPanics(t, func() {
			c.Invoke(func(d dep1) int {
				called = true
				got = d
				return d.A
			})
		})

		assert.True(t, called)
		assert.Equal(t, dep1{A: 7}, got)
	})

	t.Run("func(dep1, dep2) (type1, type2)", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)

		called := false
		got1 := dep1{}
		got2 := dep2{}
		assert.NotPanics(t, func() {
			c.Invoke(func(a dep1, b dep2) (int, string) {
				called = true
				got1 = a
				got2 = b
				return a.A, b.B
			})
		})

		assert.True(t, called)
		assert.Equal(t, dep1{A: 7}, got1)
		assert.Equal(t, dep2{B: "dep"}, got2)
	})

	t.Run("func() (type, error) nil", func(t *testing.T) {
		c := mustNew(t)
		called := false

		assert.NotPanics(t, func() {
			c.Invoke(func() (int, error) {
				called = true
				return 1, nil
			})
		})
		assert.True(t, called)
	})

	t.Run("func() (type, error) non-nil panics", func(t *testing.T) {
		c := mustNew(t)
		called := false

		assert.Panics(t, func() {
			c.Invoke(func() (int, error) {
				called = true
				return 0, errors.New("invoke failed")
			})
		})
		assert.True(t, called)
	})

	t.Run("func(dep1, dep2) (type, error) non-nil panics", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)

		called := false
		assert.Panics(t, func() {
			c.Invoke(func(_ dep1, _ dep2) (string, error) {
				called = true
				return "", errors.New("invoke dep failed")
			})
		})
		assert.True(t, called)
	})
}

func TestInvokeE_Unsupported(t *testing.T) {
	t.Run("non-function", func(t *testing.T) {
		c := mustNew(t)
		err := c.InvokeE(123)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provided argument is not a function")
	})
}

func TestInvoke_Unsupported(t *testing.T) {
	t.Run("non-function", func(t *testing.T) {
		c := mustNew(t)
		assert.Panics(t, func() {
			c.Invoke(123)
		})
	})
}

func TestGetE_ErrorStackTracePointsToCallsite(t *testing.T) {
	type missing struct{}

	c := mustNew(t)
	_, err := c.GetE[missing]()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not get type `di.missing`")
	assertFirstStackFrameContains(t, err, "TestGetE_ErrorStackTracePointsToCallsite")
}

func TestInvokeE_ErrorStackTracePointsToCallsite(t *testing.T) {
	c := mustNew(t)

	err := c.InvokeE(func() error {
		return errors.New("invokee stacktrace error")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invokee stacktrace error")
	assertFirstStackFrameContains(t, err, "TestInvokeE_ErrorStackTracePointsToCallsite")
}

func assertFirstStackFrameContains(t *testing.T, err error, wantFunc string) {
	t.Helper()

	lines := strings.Split(err.Error(), "\n")
	if !assert.GreaterOrEqual(t, len(lines), 3, "expected wrapped error with stack trace") {
		return
	}

	firstFrameFunc := lines[1]
	firstFrameFile := lines[2]

	assert.Contains(t, firstFrameFunc, wantFunc)
	assert.Contains(t, firstFrameFile, "container_test.go")
}

func TestCircularDependency_GetE_DetectedAndPrevented(t *testing.T) {
	type circularA struct{}
	type circularB struct{}

	c := mustNew(t)
	assert.NoError(t, c.Provides(func(_ circularB) circularA { return circularA{} }))
	assert.NoError(t, c.Provides(func(_ circularA) circularB { return circularB{} }))

	_, err := c.GetE[circularA]()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not get type `di.circularA`")
	assert.Contains(t, err.Error(), "circular dependency detected")
	assert.Contains(t, err.Error(), "already being loaded")
}

func TestCircularDependency_InvokeE_DetectedAndPrevented(t *testing.T) {
	type circularA struct{}
	type circularB struct{}

	c := mustNew(t)
	assert.NoError(t, c.Provides(func(_ circularB) circularA { return circularA{} }))
	assert.NoError(t, c.Provides(func(_ circularA) circularB { return circularB{} }))

	called := false
	err := c.InvokeE(func(_ circularA) {
		called = true
	})

	assert.Error(t, err)
	assert.False(t, called)
	assert.Contains(t, err.Error(), "could not invoke")
	assert.Contains(t, err.Error(), "circular dependency detected")
	assert.Contains(t, err.Error(), "already being loaded")
}

func TestMakeE_Success(t *testing.T) {
	t.Run("func() T", func(t *testing.T) {
		c := mustNew(t)
		res, err := c.MakeE[int](func() int { return 42 })
		assert.NoError(t, err)
		assert.Equal(t, 42, res)
	})

	t.Run("func(dep) T", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)
		res, err := c.MakeE[int](func(d dep1) int { return d.A * 2 })
		assert.NoError(t, err)
		assert.Equal(t, 14, res)
	})

	t.Run("func(dep1, dep2) T", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)
		res, err := c.MakeE[string](func(a dep1, b dep2) string {
			return fmt.Sprintf("%d-%s", a.A, b.B)
		})
		assert.NoError(t, err)
		assert.Equal(t, "7-dep", res)
	})

	t.Run("func() (T, error)", func(t *testing.T) {
		c := mustNew(t)
		res, err := c.MakeE[string](func() (string, error) { return "ok", nil })
		assert.NoError(t, err)
		assert.Equal(t, "ok", res)
	})

	t.Run("func(dep) (T, error)", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)
		res, err := c.MakeE[int](func(d dep1) (int, error) { return d.A + 1, nil })
		assert.NoError(t, err)
		assert.Equal(t, 8, res)
	})

	t.Run("func(dep1, dep2) (T, error)", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)
		res, err := c.MakeE[string](func(a dep1, b dep2) (string, error) {
			return fmt.Sprintf("%d-%s", a.A, b.B), nil
		})
		assert.NoError(t, err)
		assert.Equal(t, "7-dep", res)
	})
}

func TestMakeE_Failure(t *testing.T) {
	t.Run("func() (T, error) returns non-nil error", func(t *testing.T) {
		c := mustNew(t)
		_, err := c.MakeE[int](func() (int, error) {
			return 0, errors.New("result failed")
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "result failed")
	})

	t.Run("func(dep) (T, error) returns non-nil error", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)
		_, err := c.MakeE[int](func(_ dep1) (int, error) {
			return 0, errors.New("dep result failed")
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dep result failed")
	})

	t.Run("func(dep) unresolved dependency", func(t *testing.T) {
		type missing struct{ X int }
		c := mustNew(t)
		_, err := c.MakeE[int](func(_ missing) int { return 1 })
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not call result fn")
		assert.Contains(t, err.Error(), "could not find a factory for type di.missing")
	})
}

func TestMakeE_Unsupported(t *testing.T) {
	t.Run("non-function", func(t *testing.T) {
		c := mustNew(t)
		_, err := c.MakeE[int](123)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provided argument is not a function")
	})

	t.Run("func() no return values", func(t *testing.T) {
		c := mustNew(t)
		_, err := c.MakeE[int](func() {})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fn must return at least one value")
	})
}

func TestMake_Success(t *testing.T) {
	t.Run("func() T", func(t *testing.T) {
		c := mustNew(t)
		assert.NotPanics(t, func() {
			res := c.Make[int](func() int { return 99 })
			assert.Equal(t, 99, res)
		})
	})

	t.Run("func(dep) T", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)
		assert.NotPanics(t, func() {
			res := c.Make[string](func(b dep2) string { return b.B })
			assert.Equal(t, "dep", res)
		})
	})

	t.Run("func(dep1, dep2) T", func(t *testing.T) {
		c := mustNew(t)
		providesDeps(t, c)
		assert.NotPanics(t, func() {
			res := c.Make[int](func(a dep1, _ dep2) int { return a.A })
			assert.Equal(t, 7, res)
		})
	})

	t.Run("func() (T, error) nil error", func(t *testing.T) {
		c := mustNew(t)
		assert.NotPanics(t, func() {
			res := c.Make[string](func() (string, error) { return "hello", nil })
			assert.Equal(t, "hello", res)
		})
	})
}

func TestMake_Failure(t *testing.T) {
	t.Run("unresolved dependency panics", func(t *testing.T) {
		type missing struct{}
		c := mustNew(t)
		assert.Panics(t, func() {
			_ = c.Make[int](func(_ missing) int { return 1 })
		})
	})

	t.Run("func() (T, error) non-nil error panics", func(t *testing.T) {
		c := mustNew(t)
		assert.Panics(t, func() {
			_ = c.Make[int](func() (int, error) {
				return 0, errors.New("result panic")
			})
		})
	})
}

func TestMake_Unsupported(t *testing.T) {
	t.Run("non-function", func(t *testing.T) {
		c := mustNew(t)
		assert.Panics(t, func() {
			_ = c.Make[int](123)
		})
	})

	t.Run("func() no return values panics", func(t *testing.T) {
		c := mustNew(t)
		assert.Panics(t, func() {
			_ = c.Make[int](func() {})
		})
	})
}

func TestNew_NoInit(t *testing.T) {
	c, err := New()
	assert.NoError(t, err)
	assert.NotNil(t, c)
}

func TestNew_InitSuccess(t *testing.T) {
	type created struct{ V string }

	c := mustNew(t,
		func(container *Container) error {
			return container.Provides(func() created { return created{V: "ready"} })
		},
	)

	assert.Equal(t, created{V: "ready"}, mustGetE[created](t, c))
}

func TestNew_InitOrder(t *testing.T) {
	type created struct{ V int }
	order := make([]string, 0, 2)

	c := mustNew(t,
		func(container *Container) error {
			order = append(order, "register dep")
			return container.Provides(Instance(dep1{A: 21}))
		},
		func(container *Container) error {
			order = append(order, "register created")
			return container.Provides(func(d dep1) created { return created{V: d.A * 2} })
		},
	)

	assert.Equal(t, []string{"register dep", "register created"}, order)
	assert.Equal(t, created{V: 42}, mustGetE[created](t, c))
}

func TestNew_InitErrorStopsChain(t *testing.T) {
	type afterFailure struct{ V string }
	boom := errors.New("init failed")
	order := make([]string, 0, 3)
	ranThird := false

	c, err := New(
		func(container *Container) error {
			order = append(order, "first")
			return container.Provides(Instance(dep1{A: 9}))
		},
		func(container *Container) error {
			order = append(order, "second")
			return boom
		},
		func(container *Container) error {
			ranThird = true
			order = append(order, "third")
			return container.Provides(func() afterFailure { return afterFailure{V: "unreachable"} })
		},
	)

	assert.ErrorIs(t, err, boom)
	assert.NotNil(t, c)
	assert.Equal(t, []string{"first", "second"}, order)
	assert.False(t, ranThird)
	assert.Equal(t, dep1{A: 9}, mustGetE[dep1](t, c))

	_, getErr := c.GetE[afterFailure]()
	assert.Error(t, getErr)
	assert.Contains(t, getErr.Error(), "could not find a factory for type di.afterFailure")
}

func TestInstance_Helper(t *testing.T) {
	provided := &dep1{A: 33}
	c := mustNew(t, func(container *Container) error {
		return container.Provides(Instance(provided))
	})

	got := mustGetE[*dep1](t, c)
	assert.Same(t, provided, got)
}

func TestAlias_Helper(t *testing.T) {
	c := mustNew(t,
		func(container *Container) error {
			return container.Provides(Instance(aliasGreeterImpl{message: "hello"}))
		},
		func(container *Container) error {
			return container.Provides(Alias[aliasGreeterImpl, aliasGreeter]())
		},
	)

	got := mustGetE[aliasGreeter](t, c)
	assert.Equal(t, "hello", got.Greet())
}
