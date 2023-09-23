#!/bin/bash

# reset bluetooth adapter by restarting it
hciconfig hci0 down
hciconfig hci0 up

# start services
service dbus start
service bluetooth start

# wait for startup of services
msg="Waiting for services to start..."
time=0
echo -n $msg
while [[ "$(pidof start-stop-daemon)" != "" ]]; do
    sleep 1
    time=$((time + 1))
    echo -en "\r$msg $time s"
done
echo -e "\r$msg done! (in $time s)"

while true
do
  sleep 1
done
