# Elevator project - TTK4145

Elevator project for Real-time Programming TTK4145 at NTNU. This a distributed program which is designed to handle multiple elevators on a network concurrently in a fail-safe manner. 

## Dependencies

This project requires the following dependencies to be installed on the host machine:
- `elevatorserver` or `elevatorsimulator` needs to be installed and on the path of the root user
- `golang` >= 1.21 For the distributed elevator program
- `dmd` D-lang compiler for the hall request assigner

Snippets:
`sudo apt install golang`
`sudo apt install dmd`

You can also use the installation script in the `scripts/` folder in the root directory of this repository:
`chmod +x scripts/install_deps.sh`

> Note: If DMD is not in apt sources, visit https://dlang.org/download.html for dmd download

## Deployment

To deploy an instance of an elevator you only need to run:
`sudo ./scripts/create_and_start_elevatorservice.sh 1`
Where the argument specifies the ID of that instance. The script does the following:
- Make a config directory `~/.config/elevator/`
- build the elevator program and copy to `~/.config/elevator/`
- Copy the template systemd service file `elevator@.service` to `/etc/systemd/system`
- Start the `elevatorserver`
- Enable and start the service with given ID 

You can use systemctl to stop, start or view the status of a running elevator:

`sudo systemctl start elevator@1`

`sudo systemctl stop elevator@1`

`sudo systemctl status elevator@1`

## Un-deployment: Removing the elevator
To remove the elvatorservice and it's configuration files you can use the following command:

`sudo ./scripts/stop_and_remove_elevatorservice.sh 1`

"1" being the instance you want to remove. 

## Manual deployment

To deploy the elevator manually you can run the elevatorserver:

`elevatorserver --port myport`

and then run the elevator program:

`go run main.go --id myid --port myport`

> This is not part of the specification, but is useful for testing