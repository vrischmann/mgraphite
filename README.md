mgraphite
=========


[![Build Status](https://travis-ci.org/vrischmann/mgraphite.svg?branch=master)](https://travis-ci.org/vrischmann/mgraphite)
[![GoDoc](https://godoc.org/github.com/vrischmann/mgraphite?status.svg)](https://godoc.org/github.com/vrischmann/mgraphite)

mgraphite is a metrics library with reporting to a Graphite compatible server.

The design is based on what [expvar](https://golang.org/pkg/expvar) does.

usage
=====

```go
go mgr.Export(&mgr.Config{
    Interval: 5 * time.Minute,
    Addr: "localhost:2003",
})

hits := mgr.NewInt("hits")
...
hits.Add(1)
```

This will export all metrics to Graphite every 5 minutes.
