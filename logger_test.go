package llog

import (
	"net"
	"net/http"
	"sync"
	. "testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLLogErrorLogger(t *T) {
	ch := make(chan string, 1)
	s := new(http.Server)
	s.ErrorLog = newErrorLogger(LogFunc(func(msg string, kv ...KV) {
		ch <- msg
	}), KV{}, nil)
	s.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("testing")
	})
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.Serve(l)
	}()

	_, err = http.Get("http://" + l.Addr().String())
	require.Error(t, err)

	msg := <-ch
	assert.NotEmpty(t, msg)

	s.Close()
	wg.Wait()
}
