#!/bin/sh

set -ex

/usr/local/osxcross/bin/lipo \
   -create release/macos/build/trezord-arm64 release/macos/build/trezord-amd64 \
   -output release/macos/build/trezord
