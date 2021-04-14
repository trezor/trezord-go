getent group onekeyd >/dev/null || groupadd -r onekeyd
getent group plugdev >/dev/null || groupadd -r plugdev
getent passwd onekeyd >/dev/null || useradd -r -g onekeyd -d /var -s /bin/false -c "OneKey Bridge" onekeyd
usermod -a -G plugdev onekeyd
touch /var/log/onekeyd.log
chown onekeyd:onekeyd /var/log/onekeyd.log
chmod 660 /var/log/onekeyd.log
