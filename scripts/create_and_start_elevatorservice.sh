#!/bin/bash

# Check if the script is run as root
if [ "$(id -u)" != "0" ]; then
    echo "This script must be run as root."
    exit 1
fi

# Check if the number of elevators was passed as an argument
if [ -z "$1" ]; then
    echo "Please specify id (int) of the elevator."
    exit 1
fi

SERVICE_NAME="elevator@$1.service"
SERVICE_FILE_NAME="elevator@.service"
SERVICE_FILE_PATH="/etc/systemd/system"
CONFIG_FILE_PATH="/home/student/.config/elevator"

# Make dirs
mkdir -p $CONFIG_FILE_PATH/

echo "Building project and moving resources to $CONFIG_FILE_PATH"
go build -o $CONFIG_FILE_PATH/distributedElevator main.go
cp scripts/launch_elevator.sh $CONFIG_FILE_PATH
cp elevatorAlgorithm/hra/hall_request_assigner $CONFIG_FILE_PATH
echo "Installing systemd service $SERVICE_NAME"
cp scripts/$SERVICE_FILE_NAME $SERVICE_FILE_PATH

# Reload systemd, enable and start the service
systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl start "$SERVICE_NAME"

echo "$SERVICE_NAME service created and started."
