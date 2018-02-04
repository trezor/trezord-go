if which systemctl > /dev/null ; then
  systemctl enable trezord.service
  systemctl start trezord.service
else
  chkconfig --add trezord || update-rc.d trezord defaults
  service trezord start
fi
