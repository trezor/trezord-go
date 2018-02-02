if which systemctl > /dev/null ; then
  systemctl stop trezord.service
  systemctl disable trezord.service
else
  service trezord stop
  chkconfig --del trezord || update-rc.d -f trezord remove
fi
