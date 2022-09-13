## How to update hidapi

Unlike libusb, we don't have any custom patches in hidapi. However, we use only some of the files.

How to update:

1. git clone https://github.com/libusb/hidapi to `~/hidapi`
2. checkout latest stable branch
3. `rm ~/trezord-go/usb/lowlevel/hidapi/c/hidapi/*.[ch] ~/trezord-go/usb/lowlevel/hidapi/c/windows/*.[ch] ~/trezord-go/usb/lowlevel/hidapi/c/mac/*.[ch]`
4. `cp ~/hidapi/hidapi/*.[ch] ~/trezord-go/usb/lowlevel/hidapi/c/hidapi`
5. `cp ~/hidapi/windows/*.[ch] ~/trezord-go/usb/lowlevel/hidapi/c/windows`
6. `cp ~/hidapi/mac/*.[ch] ~/trezord-go/usb/lowlevel/hidapi/c/mac`
7. `cp ~/hidapi/AUTHORS.txt ~/hidapi/README.md ~/trezord-go/usb/lowlevel/hidapi/c`

Note - you need to go build `go build -a` in order to "load" the new files
