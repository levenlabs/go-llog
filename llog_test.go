package llog

import (
	"bytes"
	"io/ioutil"
	"regexp"
	. "testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTruncate(t *T) {
	assert.Equal(t, "abc", Truncate("abc", 4))
	assert.Equal(t, "abc", Truncate("abc", 3))
	assert.Equal(t, "ab...", Truncate("abc", 2))
}

func TestKV(t *T) {
	var kv KV
	assert.NotNil(t, kv.Copy())
	assert.Empty(t, kv.Copy())

	kv = KV{"foo": "a"}
	kv2 := kv.Copy()
	kv["bar"] = "b"
	kv2["bar"] = "bb"
	assert.Equal(t, KV{"foo": "a", "bar": "b"}, kv)
	assert.Equal(t, KV{"foo": "a", "bar": "bb"}, kv2)

	kv = KV{"foo": "a"}
	kv2 = kv.Set("bar", "wat")
	assert.Equal(t, KV{"foo": "a"}, kv)
	assert.Equal(t, KV{"foo": "a", "bar": "wat"}, kv2)

	kv = Merge(
		KV{"foo": "aaaaa"},
		KV{"foo": "a", "bar": "bbbbb"},
		KV{"bar": "b"},
	)
	assert.Equal(t, KV{"foo": "a", "bar": "b"}, kv)
}

func TestLLog(t *T) {
	// Unfortunately due to the nature of the package all testing involving Out
	// must be syncronous
	buf := bytes.NewBuffer(make([]byte, 0, 128))
	Out = buf

	assertOut := func(expected string) {
		out, err := buf.ReadString('\n')
		require.Nil(t, err)
		assert.Equal(t, expected, out)
	}

	SetLevelFromString("INFO")
	Debug("foo")
	Info("bar")
	Warn("baz")
	Error("buz")
	Flush()
	assertOut("~ INFO -- bar\n")
	assertOut("~ WARN -- baz\n")
	assertOut("~ ERROR -- buz\n")

	SetLevelFromString("WARN")
	Debug("foo")
	Info("bar")
	Warn("baz")
	Error("buz", KV{"a": "b"})
	Flush()
	assertOut("~ WARN -- baz\n")
	assertOut("~ ERROR -- buz -- a=\"b\"\n")
}

func TestEntryPrintOut(t *T) {
	assertEntry := func(postfix string, e entry) {
		expectedRegex := regexp.MustCompile(`^~ ` + postfix + `\n$`)
		expectedRegexTS := regexp.MustCompile(`^~ \[[^\]]+\] ` + postfix + `\n$`)

		buf := bytes.NewBuffer(make([]byte, 0, 128))

		require.Nil(t, e.printOut(buf, false))
		require.Nil(t, e.printOut(buf, true))

		noTS, err := buf.ReadString('\n')
		require.Nil(t, err)
		assert.True(t, expectedRegex.MatchString(noTS), "regex: %q line: %q", expectedRegex.String(), noTS)

		withTS, err := buf.ReadString('\n')
		require.Nil(t, err)
		assert.True(t, expectedRegexTS.MatchString(withTS), "regex: %q line: %q", expectedRegexTS.String(), withTS)
	}

	e := entry{
		level: InfoLevel,
		msg:   "this is a test",
	}
	assertEntry("INFO -- this is a test", e)

	e.kvSlice = KV{}.StringSlice()
	assertEntry("INFO -- this is a test", e)

	e.kvSlice = KV{"foo": "a"}.StringSlice()
	assertEntry("INFO -- this is a test -- foo=\"a\"", e)

	e.kvSlice = KV{"foo": "a", "bar": "b"}.StringSlice()
	assertEntry("INFO -- this is a test -- bar=\"b\" foo=\"a\"", e)

	e.kvSlice = Merge(
		KV{"foo": "aaaaa"},
		KV{"foo": "a"},
		KV{"bar": "b"},
	).StringSlice()
	assertEntry("INFO -- this is a test -- bar=\"b\" foo=\"a\"", e)
}

func BenchmarkLLog(b *B) {
	Out = ioutil.Discard
	for n := 0; n < b.N; n++ {
		Info("This is a generic message", KV{"foo": "bar"})
	}
}
