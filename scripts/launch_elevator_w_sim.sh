#!/bin/bash

# Check if the number of elevators was passed as an argument
if [ -z "$1" ]; then
    echo "Please specify the number of elevators."
    exit 1
fi

N=$1 # Number of elevators
default_udp_port=4200
default_id=0

simulator_path=$(pwd)/Simulator-v2
go_program_path=$(pwd)
go_path_macos=/opt/homebrew/bin/go

# Loop to start N elevators and simulators
for ((i=0; i<N; i++)); do
    udp_port=$((default_udp_port + i)) # Calculate UDP port for main.go
    sim_udp_port=$udp_port
    id=$((default_id + i)) # Calculate ID

# Check operating system
if [ "$(uname)" == "Darwin" ]; then
        # macOS specific commands to open new terminal windows
        # Run simulator in a new terminal window
        osascript -e "tell app \"Terminal\" to do script \"echo Starting simulator with UDP port: $sim_udp_port; $simulator_path/sim_server --port $sim_udp_port\""
        # Run main.go in a new terminal window
        sleep 1
        osascript -e "tell app \"Terminal\" to do script \"cd $go_program_path && echo Using UDP port: $udp_port; echo Using ID: $id; $go_path_macos run main.go --port $udp_port --id $id\""
    elif [ "$(uname)" == "Linux" ]; then
        # Linux specific commands to open new terminal windows
        # Run simulator in a new terminal window
        gnome-terminal -- bash -c "echo Starting simulator with UDP port: $sim_udp_port; $simulator_path/SimElevatorServer --port $sim_udp_port; exec bash"
        # Run main.go in a new terminal window
        sleep 1
        gnome-terminal -- bash -c "cd $go_program_path && echo Using UDP port: $udp_port; echo Using ID: $id; go run main.go --port $udp_port --id $id; exec bash"
    else
        echo "Unknown operating system."
        exit 1
    fi
done
