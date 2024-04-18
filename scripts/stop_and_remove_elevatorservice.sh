#!/bin/bash

# Check if the number of elevators was passed as an argument
if [ -z "$1" ]; then
    echo "Please specify id (int) of the elevator."
    exit 1
fi

systemctl stop elevator@$1.service
systemctl disable elevator@$1.service

rm -f /etc/systemd/system/elevator@.service
rm -rf /home/student/.config/elevator
