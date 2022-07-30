getent group trezord >/dev/null || groupadd -r trezord
getent group plugdev >/dev/null || groupadd -r plugdev
getent passwd trezord >/dev/null || useradd -r -g trezord -d /var -s /bin/false -c "Trezor Bridge" trezord
usermod -a -G plugdev trezord
