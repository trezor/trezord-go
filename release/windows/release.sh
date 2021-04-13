#!/bin/sh

set -e

cd $(dirname $0)

TARGET=$1
VERSION=$(cat /release/build/VERSION)

INSTALLER=onekey-bridge-$VERSION-$TARGET-install.exe

cd /release/build

cp /release/onekey.nsis onekey.nsis
cp /release/onekey.ico onekey.ico

# openssl pkcs12 -in authenticode.p12 -out authenticode.crt -clcerts -nokeys
# openssl pkcs12 -in authenticode.p12 -out authenticode.key -nocerts -nodes

SIGNKEY=/release/authenticode

if [ -r $SIGNKEY.key ]; then
    for BINARY in {onekey,devcon,wdi-simple}-{32b,64b}.exe ; do
        mv $BINARY $BINARY.unsigned
        osslsigncode sign -certs $SIGNKEY.crt -key $SIGNKEY.key -n "OneKey Bridge" -i "https://onekey.so/" -h sha256 -t "http://timestamp.comodoca.com?td=sha256" -in $BINARY.unsigned -out $BINARY
        osslsigncode verify -in $BINARY
    done
fi

if [ $TARGET = win32 ]; then
    makensis -X"OutFile $INSTALLER" -X'InstallDir "$PROGRAMFILES32\OneKey Bridge"' onekey.nsis
else
    makensis -X"OutFile $INSTALLER" -X'InstallDir "$PROGRAMFILES64\OneKey Bridge"' onekey.nsis
fi

if [ -r $SIGNKEY.key ]; then
    mv $INSTALLER $INSTALLER.unsigned
    osslsigncode sign -certs $SIGNKEY.crt -key $SIGNKEY.key -n "OneKey Bridge" -i "https://onekey.so/" -h sha256 -t "http://timestamp.comodoca.com?td=sha256" -in $INSTALLER.unsigned -out $INSTALLER
    osslsigncode verify -in $INSTALLER
fi
