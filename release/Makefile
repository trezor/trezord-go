.PHONY: windows linux macos copy gpg clean

all: windows linux macos copy

linux:
	make -C linux all

macos:
	make -C macos all

windows:
	make -C windows all

copy:
	rm -rf installers
	mkdir installers
	cp macos/build/trezor-bridge*.pkg* installers
	cp windows/build/trezor-bridge*.exe* installers
	cp linux/build/*.deb linux/build/*.rpm installers
	rm -rf binaries
	mkdir binaries
	cp macos/build/trezord binaries/trezord-darwin-universal
	cp windows/build/trezord-32b.exe binaries/trezord-windows-386.exe
	cp windows/build/trezord-64b.exe binaries/trezord-windows-amd64.exe
	cp linux/build/trezord-linux-* binaries/

clean:
	rm -rf installers binaries macos/build linux/build windows/build
