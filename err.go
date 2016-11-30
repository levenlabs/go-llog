package llog

import "github.com/levenlabs/golib/errctx"

type errKVKey int

const errKV errKVKey = 1

// ErrWithKV embeds a KV into an error, returning a new error instance. If the
// error already has a KV embedded in it then the returned error will have a
// the merging of the two.
func ErrWithKV(err error, kv KV) error {
	if err == nil {
		return nil
	}
	existingKV := errctx.Get(err, errKV)
	if existingKV != nil {
		kv = Merge(existingKV.(KV), kv)
	}
	return errctx.Set(err, errKV, kv)
}

func getInnerKV(err error) KV {
	kvi := errctx.Get(err, errKV)
	if kvi == nil {
		return KV{}
	}
	return kvi.(KV)
}

// ErrKV returns a copy of the KV embedded in the error by ErrWithKV. Will
// automatically set the "err" field on that KV to as well.
func ErrKV(err error) KV {
	kv := getInnerKV(err)
	kv["err"] = err.Error()
	return kv
}
