/*
Package trezorapi is for connecting Trezor to go programs.

It automatically uses bridge, if it's available; if it isn't,
it uses low-level USB access through C libraries, that are vendored in.

Also read the docs for trezorpb and trezorpbcall.

OS support

It works on macOS, Windows, and Linux, and maybe on FreeBSD.

On macOS, everything works automatically. Windows 8+ too. FreeBSD too.

On Linux, end user of your app needs to install udev rules -
see https://wiki.trezor.io/Udev_rules - however, that is
automatically done when installing bridge.

See also this - https://github.com/trezor/trezor-common/tree/master/udev -
which makes this - https://github.com/trezor/webwallet-data/tree/master/udev -
prepackaged udev rules for redhat and debian. You can use that in some way, if you
want to add trezor support to your go app; or you can tell your users to just
install bridge and/or copy-paste the udev rules.

On Windows 7, there is a mayhem with the drivers; let's not describe that here;
the best way there is just tell your users to install bridge, where we caught all
the driver installation issues.

Android is not supported right now, sorry.

Cross-compilation should work with xgo - https://github.com/karalabe/xgo

Trezor support

This works with both T1 and TT, and also with UDP emulators of both T1 and TT.
However, as noted further, if bridge is running at local computer, emulator might
not connect.

*/
package trezorapi
