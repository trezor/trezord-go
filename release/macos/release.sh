#!/bin/sh

# In order to get the necessary files for signing and notarization, we need to do a lot of work:
# 1) register as Apple developer, get signing certificate from Apple
#    (can take months for organizations; it is fast for individuals, but costs money)
# 2) On Mac, install XCode, generate certificates with type "Developer ID Installer"
# 3) On Mac, still in XCode, right-click in on the cert and select "export", That creates installer.p12 file (with passphrase)
# 4) Put the file to installer.p12 in certs/ dir, so as `certs/installer.p12``
# 5) Put the passphrase to certs/installer.passphrase (without newline at the end)
# 6) Then do the same, but with type "Developer ID Application"
# 7) Put those to certs/app.passphrase and certs/app.p12
#
# You will need App Store Connect API. To do that:
# 8) Go to https://appstoreconnect.apple.com/access/api
# 9) Select a user
# 10) Click on "keys", "Request access", "agree", "submit", "gereate API key"
# 11) Select "Developer" role, "Generate"
# 12) you will need to save "Issuer ID", "Key ID" and the API key (which will be PEM)
# 13) now, save
#     a) issuer ID to "certs/notarization-issuer-id"
#     b) key ID to "certs/notarization-key-id"
#     c) key to "certs/notarization-authkey.p8"

# In all, there should be those 8 files in certs
# 1. installer.passphrase
# 2. installer.p12
# 3. app.passphrase
# 4. app.p12
# 5. notarization-authkey.p8
# 6. notarization-issuer-id
# 7. notarization-key-id
# 8. .empty (invisible file to keep folder in git)




# Recommended reading for flat packages:
# https://matthew-brett.github.io/docosx/flat_packages.html
# http://bomutils.dyndns.org/tutorial.html
# http://s.sudre.free.fr/Stuff/Ivanhoe/FLAT.html
# http://s.sudre.free.fr/Software/Packages/Q&A_5.html
# https://developer.apple.com/library/content/documentation/DeveloperTools/Reference/DistributionDefinitionRef/Chapters/Distribution_XML_Ref.html#//apple_ref/doc/uid/TP40005370-CH100-SW2

set -ex

PATH=$PATH:~/.cargo/bin
SIGN_APP_PASSPHRASE_F=/release/certs/app.passphrase
SIGN_APP_CERT=/release/certs/app.p12
SIGN_INSTALLER_PASSPHRASE_F=/release/certs/installer.passphrase
SIGN_INSTALLER_CERT=/release/certs/installer.p12

cd $(dirname $0)

TARGET=$1
VERSION=$(cat /release/build/VERSION)

INSTALLER=trezor-bridge-$VERSION.pkg

mkdir -p flat-uninstall/uninstall.pkg/payload
mkdir -p flat-install/install.pkg/payload/Applications/Utilities/TREZOR\ Bridge

# first, make uninstaller

rm -rf /release/build/flat-uninstall
cp -r /release/flat-uninstall /release/build
cd /release/build/flat-uninstall/uninstall.pkg

cd scripts
find . | cpio -o --format odc --owner 0:80 | gzip -c > ../Scripts-zip
cd ..
ls -l payload/
ls -l scripts/
rm -r scripts
mv Scripts-zip Scripts
cd payload
find . | cpio -o --format odc --owner 0:80 | gzip -c > ../Payload-zip
cd ..
ls -l payload/
mkbom -u 0 -g 80 payload/ Bom
rm -r payload
mv Payload-zip Payload
cd ..
sed -i s/VERSION/$VERSION/g Distribution
sed -i s/VERSION/$VERSION/g uninstall.pkg/PackageInfo
xar -cvf ../uninstall.pkg .
cd ..

rm -rf /release/build/flat-uninstall

if [ -r $SIGN_INSTALLER_PASSPHRASE_F ]; then
    rcodesign sign --p12-file $SIGN_INSTALLER_CERT --p12-password-file $SIGN_INSTALLER_PASSPHRASE_F uninstall.pkg
fi


# second, make installer and add trezord and uninstaller

rm -rf /release/build/flat-install

cp -r /release/flat-install /release/build
cd /release/build/flat-install/install.pkg
cd scripts
find . | cpio -o --format odc --owner 0:80 | gzip -c > ../Scripts-zip
cd ..
rm -r scripts
mv Scripts-zip Scripts
cd payload

if [ -r $SIGN_INSTALLER_APP_F ]; then
    rcodesign sign --p12-file $SIGN_APP_CERT --p12-password-file $SIGN_APP_PASSPHRASE_F --code-signature-flags runtime /release/build/trezord
fi

cp /release/build/trezord Applications/Utilities/TREZOR\ Bridge/
cp ../../../uninstall.pkg Applications/Utilities/TREZOR\ Bridge/

FILES=$(find . | wc -l)
KBYTES=$(du -k -s . | cut -f 1)

find . | cpio -o --format odc --owner 0:80 | gzip -c > ../Payload-zip
cd ..
mkbom -u 0 -g 80 payload/ Bom

rm -r payload
mv Payload-zip Payload
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


# sign the installer

if [ -r $SIGN_INSTALLER_PASSPHRASE_F ]; then
    rcodesign sign --p12-file $SIGN_INSTALLER_CERT --p12-password-file $SIGN_INSTALLER_PASSPHRASE_F $INSTALLER
fi

NOT_KEY_ID_F=/release/certs/notarization-key-id
NOT_ISSUER_ID_F=/release/certs/notarization-issuer-id
NOT_AUTHKEY=/release/certs/notarization-authkey.p8
if [ -r $NOT_KEY_ID_F ]; then
    NOT_KEY_ID=$(cat $NOT_KEY_ID_F)
    NOT_ISSUER_ID=$(cat $NOT_ISSUER_ID_F)
    NOT_JSON=not-data.json
    rcodesign encode-app-store-connect-api-key -o $NOT_JSON $NOT_ISSUER_ID $NOT_KEY_ID $NOT_AUTHKEY
    rcodesign notary-submit \
        --api-key-path $NOT_JSON \
        --staple \
        $INSTALLER
    rm $NOT_JSON
fi
