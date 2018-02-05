#!/bin/sh

set -e

cd $(dirname $0)

GPGSIGNKEY=86E6792FC27BFD478860C11091F3B339B9A02A3D
TARGET=$1
VERSION=$(cat /release/build/VERSION)

cd /release/build

install -D -m 0755 trezord                  ./usr/bin/trezord
install -D -m 0644 /release/trezor.rules    ./lib/udev/rules.d/51-trezor.rules
install -D -m 0644 /release/trezord.service ./usr/lib/systemd/system/trezord.service

# prepare GPG signing environment
GPG_PRIVKEY=/release/privkey.asc
if [ -r $GPG_PRIVKEY ]; then
    export GPG_TTY=$(tty)
    export LC_ALL=en_US.UTF-8
    gpg --import /release/privkey.asc
    GPG_SIGN=gpg
fi

NAME=trezor-bridge

rm -f  *.tar.bz2
tar -cjf $NAME-$VERSION.tar.bz2 ./usr ./lib

for TYPE in "deb" "rpm"; do
	case "$TARGET-$TYPE" in
		lin32-deb)
			ARCH=i386
			DEPS=""
			;;
		lin64-deb)
			ARCH=amd64
			DEPS=""
			;;
		lin32-rpm)
			ARCH=i386
			DEPS=""
			;;
		lin64-rpm)
			ARCH=x86_64
			DEPS=""
			;;
	esac
	fpm \
		-s tar \
		-t $TYPE \
		-a $ARCH \
		-n $NAME \
		-v $VERSION \
		--license "LGPL-3.0" \
		--vendor "SatoshiLabs" \
		--description "Communication daemon for TREZOR" \
		--maintainer "SatoshiLabs <stick@satoshilabs.com>" \
		--url "https://trezor.io/" \
		--category "Productivity/Security" \
		--before-install /release/fpm.before-install.sh \
		--after-install /release/fpm.after-install.sh \
		--before-remove /release/fpm.before-remove.sh \
		$DEPS \
		$NAME-$VERSION.tar.bz2
	case "$TYPE-$GPG_SIGN" in
		deb-gpg)
			/release/dpkg-sig -k $GPGSIGNKEY --sign builder trezor-bridge_${VERSION}_${ARCH}.deb
			;;
		rpm-gpg)
			rpm --addsign -D "%_gpg_name $GPGSIGNKEY" trezor-bridge-${VERSION}-1.${ARCH}.rpm
			;;
	esac
done


rm -rf ./usr ./lib
