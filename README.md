trezord-go
===

```
go build
./trezord-go -h
```

Quick guide to cross-compiling
----
Prerequisities:

* install go
* `go get github.com/jpochyla/trezord-go`
* install docker
* `docker pull karalabe/xgo-latest`
* `go get github.com/karalabe/xgo`

Then:
* `cd ~/go/src/github.com/jpochyla/trezord-go`
* `xgo --targets=windows/amd64,windows/386,darwin/amd64,darwin/386,linux/amd64,linux/386,linux/arm-5,linux/arm-6,linux/arm-7,linux/arm64,linux/mips64,linux/mips64le,linux/mips,linux/mipsle`
 or any subset of the targets
