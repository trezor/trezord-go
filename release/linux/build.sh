#!/bin/sh

set -ex
PATH=$PATH:/usr/local/go/bin

git config --global --add safe.directory /trezord

case $TREZORD_BUILD in
  "go-arm64-musl")
    CGO_ENABLED=1 CC="/usr/local/musl/bin/aarch64-unknown-linux-musl-gcc" GOARCH=arm64 \
        go build --ldflags '-linkmode external -extldflags "-static" -extld /usr/local/musl/bin/aarch64-unknown-linux-musl-gcc' \
        -o release/linux/build/trezord-linux-arm64
    ;;

  "go-386-musl")
     CGO_ENABLED=1 CC="/usr/local/musl/bin/i686-unknown-linux-musl-gcc" GOARCH=386 \
        go build --ldflags '-linkmode external -extldflags "-static" -extld /usr/local/musl/bin/i686-unknown-linux-musl-gcc' \
        -o release/linux/build/trezord-linux-386
    ;;

  "go-amd64-musl")
     CGO_ENABLED=1 CC="/usr/local/musl/bin/x86_64-unknown-linux-musl-gcc" GOARCH=amd64 \
        go build --ldflags '-linkmode external -extldflags "-static" -extld /usr/local/musl/bin/x86_64-unknown-linux-musl-gcc' \
        -o release/linux/build/trezord-linux-amd64
    ;;

  *)
    echo -n "unknown build"
    exit 1
    ;;
esac