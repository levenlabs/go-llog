package llog

import (
	"context"
	"fmt"

	"github.com/levenlabs/errctx"
)

type kvKey int

// ErrWithKV embeds the merging of a set of KVs into an error and Marks the
// function for convenience, returning a new error instance. If the error
// already has a KV embedded in it then the returned error will have the
// merging of them all.
func ErrWithKV(err error, kvs ...KV) error {
	if err == nil {
		return nil
	}
	kv := Merge(kvs...)
	existingKV := errctx.Get(err, kvKey(0))
	if existingKV != nil {
		kv = Merge(existingKV.(KV), kv)
	}
	return errctx.MarkSkip(errctx.Set(err, kvKey(0), kv), 1)
}

// ErrKV returns a copy of the KV embedded in the error by ErrWithKV as well as
// any line from errctx.Mark as the key "source" if "source" wasn't already set.
// Returns empty KV if no KV was previously embedded and no line was marked.
// Will automatically set the "err" field on the returned KV as well.
func ErrKV(err error) KV {
	if err == nil {
		return KV{}
	}
	var kv KV
	kvi := errctx.Get(err, kvKey(0))
	if kvi != nil {
		kv = kvi.(KV)
	}
	// look at the base in case it's already an errctx just so we don't end up
	// showing the errType as errctx.errctx all the time
	errBase := errctx.Base(err)
	errStr := errBase.Error()
	// try to better handle empty errors
	if errStr == "" {
		// if the error is a stringer, try to get the string from that
		if stringer, ok := errBase.(fmt.Stringer); ok {
			errStr = stringer.String()
		}
		// otherwise fallback to printing the go representation
		if errStr == "" {
			errStr = fmt.Sprintf("%#v", errBase)
		}
	}
	newKV := KV{
		"err":     errStr,
		"errType": fmt.Sprintf("%T", errBase),
	}
	if line, ok := errctx.Line(err); ok && kv["source"] == nil {
		newKV["source"] = line
	}
	if kv == nil {
		return newKV
	}
	return Merge(kv, newKV)
}

// CtxWithKV embeds a KV into a Context, returning a new Context instance. If
// the Context already has a KV embedded in it then the returned context's KV
// will be the merging of the two.
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
