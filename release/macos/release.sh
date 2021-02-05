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

mkdir -p flat-uninstall/uninstall.pkg/payload-prev
# mkdir -p flat-install/install.pkg/payload-prev/Applications/Utilities/TREZOR\ Bridge

# first, make uninstaller

rm -rf /release/build/flat-uninstall
cp -r /release/flat-uninstall /release/build
cd /release/build/flat-uninstall/uninstall.pkg

cd scripts-prev
find . | cpio -o --format odc --owner 0:80 | gzip -c > ../Scripts
cd ..
rm -r scripts-prev
cd payload-prev
find . | cpio -o --format odc --owner 0:80 | gzip -c > ../Payload
cd ..
mkbom -u 0 -g 80 payload-prev/ Bom
rm -r payload-prev
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
cd scripts-prev
find . | cpio -o --format odc --owner 0:80 | gzip -c > ../Scripts
cd ..
rm -r scripts-prev
cd payload-prev

cp /release/build/trezord Applications/Utilities/TREZOR\ Bridge/
cp ../../../uninstall.pkg Applications/Utilities/TREZOR\ Bridge/

FILES=$(find . | wc -l)
KBYTES=$(du -k -s . | cut -f 1)

find . | cpio -o --format odc --owner 0:80 | gzip -c > ../Payload
cd ..
mkbom -u 0 -g 80 payload-prev/ Bom

rm -r payload-prev
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

# In order to get the necessary files for signing, do the following:
# 1) register as Apple developer, get signing certificate from Apple (can take months)
# 2) On Mac, install XCode, generate certificates with type "Developer ID Installer"
# 3) On Mac, still in XCode, right-click in on the cert and select "export", That creates cert.p12 file (with passphrase)
# 4) On Mac, do `productsign --sign 'Developer ID Installer: YourDeveloperId' 'unsigned-package.pkg' 'signed-package.pkg'`
#     where YourDeveloperId is your developer ID and unsigned-package is an unsigned package (this needs to be done only once)
# 5) Copy the cert.p12 and signed-package.pkg to Linux
# 6) On linux, do `mkdir certs ; xar -f signed-package.pkg --extract-certs certs`, that puts cert00, cert01 and cert02 in the certs dir
# 7) On linux, do `openssl pkcs12 -in cert.p12 -nodes | openssl rsa -out key.pem`, that creates key.pem

# The cert00, cert01 and cert02 files can now be used for pkg signing.


# sign the installer
PRIVKEY=/release/key.pem
if [ -r $PRIVKEY ]; then
    SIGNLEN=$(: | openssl dgst -sign $PRIVKEY -binary | wc -c)
    xar --sign -f $INSTALLER --digestinfo-to-sign digestinfo.dat \
        --sig-size $SIGNLEN \
        --cert-loc /release/cert00 \
        --cert-loc /release/cert01 \
        --cert-loc /release/cert02
    openssl rsautl -sign -inkey $PRIVKEY -in digestinfo.dat -out signature.dat
    xar --inject-sig signature.dat -f $INSTALLER
    rm -f signature.dat digestinfo.dat
fi
