all: .windows .linux .mac .copy

.linux:
	make -C linux all

.mac:
	make -C mac all

.windows:
	make -C windows all

.copy:
	rm -rf installers
	mkdir installers
	cp mac/build/*.pkg installers
	cp windows/build/trezor-bridge*.exe installers
	cp linux/build/*.deb linux/build/*.rpm installers
