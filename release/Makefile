.PHONY: windows linux macos copy gpg clean

all: windows linux macos copy gpg rename

linux:
	make -C linux all

macos:
	make -C macos all

windows:
	make -C windows all

copy:
	rm -rf installers
	mkdir installers
	cp macos/build/*.pkg installers
	cp windows/build/trezor-bridge*.exe installers
	cp linux/build/*.deb linux/build/*.rpm installers

gpg:
	# gpg -u 54067D8BBF00554181B5AB8F26A3A56662F0E7E2 --armor --detach-sig installers/trezor-bridge*.exe
	# gpg -u 54067D8BBF00554181B5AB8F26A3A56662F0E7E2 --armor --detach-sig installers/*.pkg

clean:
	rm -rf installers

rename:
	cd installers && for f in trezor*;do mv $$f onekey$$(echo $$f | cut -c7-); done;
