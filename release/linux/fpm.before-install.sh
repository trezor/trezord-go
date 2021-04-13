getent group onekey >/dev/null || groupadd -r onekey
getent group plugdev >/dev/null || groupadd -r plugdev
getent passwd onekey >/dev/null || useradd -r -g onekey -d /var -s /bin/false -c "OneKey Bridge" onekey
usermod -a -G plugdev onekey
touch /var/log/onekey.log
chown onekey:onekey /var/log/onekey.log
chmod 660 /var/log/onekey.log
