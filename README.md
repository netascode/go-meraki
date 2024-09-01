[![Tests](https://github.com/netascode/go-meraki/actions/workflows/test.yml/badge.svg)](https://github.com/netascode/go-meraki/actions/workflows/test.yml)

# go-meraki

`go-meraki` is a Go client library for Cisco Meraki. It is based on Nathan's excellent [goaci](https://github.com/brightpuddle/goaci) module and features a simple, extensible API and [advanced JSON manipulation](#result-manipulation).

## Getting Started

### Installing

To start using `go-meraki`, install Go and `go get`:

`$ go get -u github.com/netascode/go-meraki`

### Basic Usage

```go
package main

import "github.com/netascode/go-meraki"

func main() {
    client, _ := meraki.NewClient("abc123")

    res, _ := client.Get("/organizations")
    println(res.Get("0.name").String())
}
```

This will print something like:

```
My First Organization
```

#### Result manipulation

`meraki.Result` uses GJSON to simplify handling JSON results. See the [GJSON](https://github.com/tidwall/gjson) documentation for more detail.

```go
res, _ := client.Get("/organziations")

for _, obj := range res.Array() {
    println(obj.Get("@pretty").String()) // pretty print network objects
}
```

#### POST data creation

`meraki.Body` is a wrapper for [SJSON](https://github.com/tidwall/sjson). SJSON supports a path syntax simplifying JSON creation.

```go
body := meraki.Body{}.
    Set("name", "NewNetwork1").
    Set("productTypes", []string{"switch"})
client.Post("/organizations/123456/networks", body.Str)
```

## Documentation

See the [documentation](https://godoc.org/github.com/netascode/go-meraki) for more details.
