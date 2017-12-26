[![GoDoc](https://godoc.org/github.com/deadsy/libusb?status.svg)](https://godoc.org/github.com/deadsy/libusb)

# libusb
golang wrapper for libusb-1.0

The API for libusb has been mapped 1-1 to equivalent go functions and types.

See http://libusb.info/ for more information on the C-API

## Wrapper Status

Per the libusb API categories

 * Library initialization/deinitialization: complete
 * Device handling and enumeration: complete
 * Miscellaneous: complete
 * USB descriptors: complete
 * Device hotplug event notification: not started
 * Asynchronous device I/O: not started
 * Polling and timing: not started
 * Synchronous device I/O: complete

## Maturity

Alpha: Some testing. No known bugs.
