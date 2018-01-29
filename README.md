trezord-go
===

```
go build
./trezord-go -h
```

Quick guide to cross-compiling
----

Prerequisites:

* `go get github.com/karalabe/xgo`
* `docker pull karalabe/xgo-latest`

Compiling for officially supported platforms:

* `$GOPATH/bin/xgo -targets=windows/amd64,windows/386,darwin/amd64,linux/amd64,linux/386 github.com/trezor/trezord-go`
