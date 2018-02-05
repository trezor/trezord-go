getent group trezord >/dev/null || groupadd -r trezord
getent group plugdev >/dev/null || groupadd -r plugdev
getent passwd trezord >/dev/null || useradd -r -g trezord -d /var -s /sbin/nologin -c "TREZOR Bridge" trezord
usermod -a -G plugdev trezord
touch /var/log/trezord.log
chown trezord:trezord /var/log/trezord.log
chmod 660 /var/log/trezord.log
