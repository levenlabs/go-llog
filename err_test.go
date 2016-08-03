package llog

import (
	"errors"
	. "testing"

	"github.com/stretchr/testify/assert"
)

func TestErrKV(t *T) {
	err := errors.New("foo")
	assert.Equal(t, KV{"err": err}, ErrKV(err))

	kv := KV{"a": "a"}
	err2 := ErrWithKV(err, kv)
	assert.Equal(t, KV{"err": err}, ErrKV(err))
	assert.Equal(t, KV{"err": err2, "a": "a"}, ErrKV(err2))

	// changing the kv now shouldn't do anything
	kv["a"] = "b"
	assert.Equal(t, KV{"err": err}, ErrKV(err))
	assert.Equal(t, KV{"err": err2, "a": "a"}, ErrKV(err2))

	// a new ErrWithKV shouldn't affect the previous one
	err3 := ErrWithKV(err2, KV{"b": "b"})
	assert.Equal(t, KV{"err": err}, ErrKV(err))
	assert.Equal(t, KV{"err": err2, "a": "a"}, ErrKV(err2))
	assert.Equal(t, KV{"err": err3, "a": "a", "b": "b"}, ErrKV(err3))

	// make sure precedence works
	err4 := ErrWithKV(err3, KV{"b": "bb"})
	assert.Equal(t, KV{"err": err}, ErrKV(err))
	assert.Equal(t, KV{"err": err2, "a": "a"}, ErrKV(err2))
	assert.Equal(t, KV{"err": err3, "a": "a", "b": "b"}, ErrKV(err3))
	assert.Equal(t, KV{"err": err4, "a": "a", "b": "bb"}, ErrKV(err4))
}
