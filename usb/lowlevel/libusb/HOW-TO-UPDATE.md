## How to update libusb

1. clone libusb somewhere from official repo (say `~/libusb`)
2. `git checkout` to the latest stable version of libusb we used (to get patchset)
3. get list of current patches

   ```
   diff -ur  ~/libusb/libusb ~/trezord-go/usb/lowlevel/libusb/c/ > ~/trezord-patchset.diff
   ```
4. checkout the latest stable libusb
5. `mv ~/trezord-go/usb/lowlevel/libusb/c/ ~/trezord-go/usb/lowlevel/libusb/c_old`
6. `cp -r ~/libusb/libusb ~/trezord-go/usb/lowlevel/libusb/c`
7. `cp ~/libusb/AUTHORS ~/libusb/COPYING ~/trezord-go/usb/lowlevel/libusb/c`
8. `cp ~/trezord-go/usb/lowlevel/libusb/c_old/config.h ~/trezord-go/usb/lowlevel/libusb/c`
9. try to apply the patches from ~/trezord-patchset.diff to the new code (either manually or automatically)
11. delete unusued files, so far:

    ```
    usb/lowlevel/libusb/c/Makefile*
    usb/lowlevel/libusb/c/libusb-1.0.*
    usb/lowlevel/libusb/c/os/haiku*
    usb/lowlevel/libusb/c/os/linux_udev.c
    usb/lowlevel/libusb/c/os/netbsd_usb.c
    usb/lowlevel/libusb/c/os/null_usb.c
    usb/lowlevel/libusb/c/os/openbsd_usb.c
    usb/lowlevel/libusb/c/os/sunos_usb.c
    usb/lowlevel/libusb/c/os/sunos_usb.h
    ```

12. `rm -r ~/trezord-go/usb/lowlevel/libusb/c_old` when all is working fine

Note - you need to go build `go build -a` in order to "load" the new files
