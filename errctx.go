package llog

import (
	"context"

	"github.com/levenlabs/golib/errctx"
)

type kvKey int

// ErrWithKV embeds the merging of a set of KVs into an error, returning a new
// error instance. If the error already has a KV embedded in it then the
// returned error will have the merging of them all.
func ErrWithKV(err error, kvs ...KV) error {
	if err == nil {
		return nil
	}
	kv := Merge(kvs...)
	existingKV := errctx.Get(err, kvKey(0))
	if existingKV != nil {
		kv = Merge(existingKV.(KV), kv)
	}
	return errctx.Set(err, kvKey(0), kv)
}

// ErrKV returns a copy of the KV embedded in the error by ErrWithKV. Returns
// empty KV if no KV was previously embedded. Will automatically set the "err"
// field on the returned KV as well.
func ErrKV(err error) KV {
	var kv KV
	if kvi := errctx.Get(err, kvKey(0)); kvi == nil {
		kv = KV{}
	} else {
		kv = kvi.(KV)
	}
	kv["err"] = err.Error()
	return kv
}

// CtxWithKV embeds a KV into a Context, returning a new Context instance. If
// the Context already has a KV embedded in it then the returned error will have
// the merging of the two.
func CtxWithKV(ctx context.Context, kvs ...KV) context.Context {
	kv := Merge(kvs...)
	existingKV := ctx.Value(kvKey(0))
	if existingKV != nil {
		kv = Merge(existingKV.(KV), kv)
	}
	return context.WithValue(ctx, kvKey(0), kv)
}

// CtxKV returns a copy of the KV embedded in the Context by CtxWithKV
func CtxKV(ctx context.Context) KV {
	kv := ctx.Value(kvKey(0))
	if kv == nil {
		return KV{}
	}
	return kv.(KV)
}
