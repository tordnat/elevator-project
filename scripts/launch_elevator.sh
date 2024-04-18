#!/bin/bash

# Check if the number of elevators was passed as an argument
if [ -z "$1" ]; then
    echo "Please specify id (int) of the elevator."
    exit 1
fi

default_udp_port=15657
id=$1
elevator_server_path=elevatorserver
go_program_path="/home/student/.config/elevator"

udp_port=$((default_udp_port + id)) # Calculate UDP port for main.go
elevator_server_udp_port=$udp_port


# Check operating system
if [ "$(uname)" == "Linux" ]; then
        echo "Starting elevatorserver with UDP port: $default_udp_port" 
	elevatorserver --port $default_udp_port &
        sleep 1
        echo "Starting elevator using UDP port: $default_udp_port ID: $id" 
	cd $go_program_path
	./distributedElevator --port $default_udp_port --id $id
else
        echo "Unknown operating system."
        exit 1
fi
