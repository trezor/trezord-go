#!/bin/sh

set -e

cd $(dirname $0)

TARGET=$1
VERSION=$(cat /release/build/VERSION)

INSTALLER=trezor-bridge-$VERSION-$TARGET-install.exe

cd /release/build

cp /release/trezord.nsis trezord.nsis
cp /release/trezord.ico trezord.ico

SIGNKEY=/release/authenticode

if [ -r $SIGNKEY.der ]; then
    for BINARY in {trezord,devcon,wdi-simple}-{32b,64b}.exe ; do
        mv $BINARY $BINARY.unsigned
        osslsigncode sign -certs $SIGNKEY.p7b -key $SIGNKEY.der -n "Trezor Bridge" -i "https://trezor.io/" -h sha256 -t "http://timestamp.comodoca.com?td=sha256" -in $BINARY.unsigned -out $BINARY
        osslsigncode verify -in $BINARY
    done
fi

if [ $TARGET = win32 ]; then
    makensis -X"OutFile $INSTALLER" -X'InstallDir "$PROGRAMFILES32\TREZOR Bridge"' trezord.nsis
else
    makensis -X"OutFile $INSTALLER" -X'InstallDir "$PROGRAMFILES64\TREZOR Bridge"' trezord.nsis
fi

if [ -r $SIGNKEY.der ]; then
    mv $INSTALLER $INSTALLER.unsigned
    osslsigncode sign -certs $SIGNKEY.p7b -key $SIGNKEY.der -n "TREZOR Bridge" -i "https://trezor.io/" -h sha256 -t "http://timestamp.comodoca.com?td=sha256" -in $INSTALLER.unsigned -out $INSTALLER
    osslsigncode verify -in $INSTALLER
fi
