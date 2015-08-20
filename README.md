# go-llog

The leven logging library

[![GoDoc](https://godoc.org/github.com/levenlabs/go-llog?status.svg)](https://godoc.org/github.com/levenlabs/go-llog)

A simple general purpose logging library with multiple logging levels

## Goals

The goal in llog is to write logs which have a general, static error message
attached to them, describing what kind of an message happened and at what step
it happened, but leaving out any context about the message (such as which user
caused it, or what a particular parameter to a function was). The context is
then provided in the form of optional key/value pairs, which can then be easily
parsed by something like elk.

The benefits of this system is that it makes it very easy to categorize
messages, and also very easy to scan *all* messages across services for a
particular pattern. For instance, if you always include the IP address in the
key/value set across all projects, you can very easily determine all the actions
taken by a particular IP address.

## Usage

```
import "github.com/levenlabs/go-llog"

func main() {
    llog.Info("Here's a generic log message!")
    llog.Error("an error happened", llog.KV{"userID":1111, "err": err, "sky": "blue"})
}
```

These will output:

```
~ INFO -- Here's a generic log message!
~ ERROR -- an error happened -- userID=1111 err=some-error sky=blue
```
