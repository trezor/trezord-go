#!/bin/sh

set -e

cd $(dirname $0)

TARGET=$1
VERSION=$(cat /release/build/VERSION)

INSTALLER=trezor-bridge-$VERSION-$TARGET-install.exe

cd /release/build

cp /release/trezord.nsis trezord.nsis
cp /release/trezord.ico trezord.ico

# To sign for Windows, put code signing key and signature
# to authenticode.key and authenticode.crt in this folder

SIGNKEY=/release/authenticode

if [ -r $SIGNKEY.key ]; then
    for BINARY in {trezord,devcon,wdi-simple}-{32b,64b}.exe ; do
        mv $BINARY $BINARY.unsigned
        osslsigncode sign -certs $SIGNKEY.crt -key $SIGNKEY.key -n "Trezor Bridge" -i "https://trezor.io/" -h sha384 -t "http://timestamp.comodoca.com?td=sha384" -in $BINARY.unsigned -out $BINARY
        osslsigncode verify -in $BINARY
    done
fi

if [ $TARGET = win32 ]; then
    makensis -X"OutFile $INSTALLER" -X'InstallDir "$PROGRAMFILES32\TREZOR Bridge"' trezord.nsis
else
    makensis -X"OutFile $INSTALLER" -X'InstallDir "$PROGRAMFILES64\TREZOR Bridge"' trezord.nsis
fi

if [ -r $SIGNKEY.key ]; then
    mv $INSTALLER $INSTALLER.unsigned
    osslsigncode sign -certs $SIGNKEY.crt -key $SIGNKEY.key -n "Trezor Bridge" -i "https://trezor.io/" -h sha384 -t "http://timestamp.comodoca.com?td=sha384" -in $INSTALLER.unsigned -out $INSTALLER
    osslsigncode verify -in $INSTALLER
fi

GPG_PRIVKEY=/release/gpg-privkey.asc
echo 1
if [ -r $GPG_PRIVKEY ]; then
    echo 2
    export GPG_TTY=$(tty)
    export LC_ALL=C.UTF-8
    gpg --import /release/privkey.asc
    GPGSIGNKEY=$(gpg --list-keys --with-colons | grep '^pub' | cut -d ":" -f 5)
    gpg -u $GPGSIGNKEY --armor --detach-sig $INSTALLER
fi
