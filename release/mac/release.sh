#!/bin/sh

# Recommended reading for flat packages:
# https://matthew-brett.github.io/docosx/flat_packages.html
# http://bomutils.dyndns.org/tutorial.html
# http://s.sudre.free.fr/Stuff/Ivanhoe/FLAT.html
# http://s.sudre.free.fr/Software/Packages/Q&A_5.html
# https://developer.apple.com/library/content/documentation/DeveloperTools/Reference/DistributionDefinitionRef/Chapters/Distribution_XML_Ref.html#//apple_ref/doc/uid/TP40005370-CH100-SW2

set -ex

cd $(dirname $0)

TARGET=$1
VERSION=$(cat /release/build/VERSION)

INSTALLER=trezor-bridge-$VERSION.pkg

# first, make uninstaller

rm -rf /release/build/flat-uninstall
cp -r /release/flat-uninstall /release/build
cd /release/build/flat-uninstall/uninstall.pkg

cd scripts
find . | cpio -o --format odc --owner 0:80 | gzip -c > ../Scripts
cd ..
rm -r scripts
cd payload
find . | cpio -o --format odc --owner 0:80 | gzip -c > ../Payload
cd ..
mkbom -u 0 -g 80 payload/ Bom
rm -r payload
cd ..
sed -i s/VERSION/$VERSION/g Distribution
sed -i s/VERSION/$VERSION/g uninstall.pkg/PackageInfo
xar -cvf ../uninstall.pkg .
cd ..

rm -rf /release/build/flat-uninstall

# second, make installer and add trezord and uninstaller

rm -rf /release/build/flat-install

cp -r /release/flat-install /release/build
cd /release/build/flat-install/install.pkg
cd scripts
find . | cpio -o --format odc --owner 0:80 | gzip -c > ../Scripts
cd ..
rm -r scripts
cd payload

rm Applications/Utilities/TREZOR\ Bridge/trezord-fake
cp /release/build/trezord Applications/Utilities/TREZOR\ Bridge/
cp ../../../uninstall.pkg Applications/Utilities/TREZOR\ Bridge/

FILES=$(find . | wc -l)
KBYTES=$(du -k -s . | cut -f 1)

find . | cpio -o --format odc --owner 0:80 | gzip -c > ../Payload
cd ..
mkbom -u 0 -g 80 payload/ Bom

rm -r payload
cd ..
sed -i s/VERSION/$VERSION/g Distribution
sed -i s/KBYTES/$KBYTES/g Distribution
sed -i s/VERSION/$VERSION/g install.pkg/PackageInfo
sed -i s/FILES/$FILES/g install.pkg/PackageInfo
sed -i s/KBYTES/$KBYTES/g install.pkg/PackageInfo

xar -cvf ../install.pkg .

cd ..
rm -r /release/build/flat-install
rm uninstall.pkg

mv install.pkg $INSTALLER
