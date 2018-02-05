# trezord-go

[![Build Status](https://travis-ci.org/trezor/trezord-go.svg?branch=master)](https://travis-ci.org/trezor/trezord-go) [![gitter](https://badges.gitter.im/trezor/community.svg)](https://gitter.im/trezor/community)

TREZOR Communication Daemon aka TREZOR Bridge (written in Go)

```
go build
./trezord-go -h
```

## Guide to compiling packages

Prerequisites:

* `go get github.com/karalabe/xgo`
* `docker pull karalabe/xgo-latest`
* make sure `xgo` and `docker` are in `$PATH`
* `cd release && make all`; the installers are in `installers`

## Quick guide to cross-compiling

Prerequisites:

* `go get github.com/karalabe/xgo`
* `docker pull karalabe/xgo-latest`

Compiling for officially supported platforms:

* `$GOPATH/bin/xgo -targets=windows/amd64,windows/386,darwin/amd64,linux/amd64,linux/386 github.com/trezor/trezord-go`
