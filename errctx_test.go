package llog

import (
	"context"
	"errors"
	. "testing"

	"github.com/stretchr/testify/assert"
)

func TestErrKV(t *T) {
	err := errors.New("foo")
	assert.Equal(t, KV{"err": err.Error()}, ErrKV(err))

	kv := KV{"a": "a"}
	err2 := ErrWithKV(err, kv)
	assert.Equal(t, KV{"err": err.Error()}, ErrKV(err))
	assert.Equal(t, KV{"err": err2.Error(), "a": "a", "source": "errctx_test.go:16"}, ErrKV(err2))

	// changing the kv now shouldn't do anything
	kv["a"] = "b"
	assert.Equal(t, KV{"err": err.Error()}, ErrKV(err))
	assert.Equal(t, KV{"err": err2.Error(), "a": "a", "source": "errctx_test.go:16"}, ErrKV(err2))

	// a new ErrWithKV shouldn't affect the previous one
	err3 := ErrWithKV(err2, KV{"b": "b"})
	assert.Equal(t, KV{"err": err.Error()}, ErrKV(err))
	assert.Equal(t, KV{"err": err2.Error(), "a": "a", "source": "errctx_test.go:16"}, ErrKV(err2))
	assert.Equal(t, KV{"err": err3.Error(), "a": "a", "b": "b", "source": "errctx_test.go:16"}, ErrKV(err3))

	// make sure precedence works
	err4 := ErrWithKV(err3, KV{"b": "bb"})
	assert.Equal(t, KV{"err": err.Error()}, ErrKV(err))
	assert.Equal(t, KV{"err": err2.Error(), "a": "a", "source": "errctx_test.go:16"}, ErrKV(err2))
	assert.Equal(t, KV{"err": err3.Error(), "a": "a", "b": "b", "source": "errctx_test.go:16"}, ErrKV(err3))
	assert.Equal(t, KV{"err": err4.Error(), "a": "a", "b": "bb", "source": "errctx_test.go:16"}, ErrKV(err4))

	err = nil
	assert.Equal(t, KV{}, ErrKV(err))
}

func TestCtxKV(t *T) {
	ctx := context.Background()
	assert.Equal(t, KV{}, CtxKV(ctx))

	kv := KV{"a": "a"}
	ctx2 := CtxWithKV(ctx, kv)
	assert.Equal(t, KV{}, CtxKV(ctx))
	assert.Equal(t, KV{"a": "a"}, CtxKV(ctx2))

	// changing the kv now shouldn't do anything
	kv["a"] = "b"
	assert.Equal(t, KV{}, CtxKV(ctx))
	assert.Equal(t, KV{"a": "a"}, CtxKV(ctx2))

	// a new CtxWithKV shouldn't affect the previous one
	ctx3 := CtxWithKV(ctx2, KV{"b": "b"})
	assert.Equal(t, KV{}, CtxKV(ctx))
	assert.Equal(t, KV{"a": "a"}, CtxKV(ctx2))
	assert.Equal(t, KV{"a": "a", "b": "b"}, CtxKV(ctx3))

	// make sure precedence works
	ctx4 := CtxWithKV(ctx3, KV{"b": "bb"})
	assert.Equal(t, KV{}, CtxKV(ctx))
	assert.Equal(t, KV{"a": "a"}, CtxKV(ctx2))
	assert.Equal(t, KV{"a": "a", "b": "b"}, CtxKV(ctx3))
	assert.Equal(t, KV{"a": "a", "b": "bb"}, CtxKV(ctx4))
}
