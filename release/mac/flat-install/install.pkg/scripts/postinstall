#!/bin/sh

set -x
set -e

# find out which user is running the installation
inst_user=`stat /dev/console | cut -f 5 -d ' '`

# load the agent file into launchd with correct user

agent_file=/Library/LaunchAgents/com.bitcointrezor.trezorBridge.trezord.plist

set +e
sudo -u $inst_user launchctl unload $agent_file
set -e
sudo -u $inst_user launchctl load $agent_file
