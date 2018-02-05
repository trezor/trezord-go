all: .windows .linux .macos .copy

.linux:
	make -C linux all

.macos:
	make -C macos all

.windows:
	make -C windows all

.copy:
	rm -rf installers
	mkdir installers
	cp macos/build/*.pkg installers
	cp windows/build/trezor-bridge*.exe installers
	cp linux/build/*.deb linux/build/*.rpm installers

.gpg:
	gpg --armor --detach-sig installers/trezor-bridge*.exe
	gpg --armor --detach-sig installers/*.pkg
