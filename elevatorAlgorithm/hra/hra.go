package hra

import (
	"elevatorAlgorithm/elevator"
	"elevatorDriver/elevio"
	"encoding/json"
	"fmt"

	//"encoding/json"
	"os/exec"
)

// TODO: Move this to a different file
type LocalElevatorState struct {
	Behaviour   elevator.ElevatorBehaviour
	Floor       int
	Direction   elevio.MotorDirection
	CabRequests []bool
}

type ElevatorSystem struct {
	HallRequests   [][]bool
	ElevatorStates map[string]LocalElevatorState
}

type hraLocalElevatorState struct {
	Behaviour   string `json:"behaviour"` // "idle", "moving", or "doorOpen"
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"` // "up", "down", or "stop"
	CabRequests []bool `json:"cabRequests"`
}

type hraElevatorSystem struct {
	HallRequests   [][]bool                         `json:"hallRequests"`
	ElevatorStates map[string]hraLocalElevatorState `json:"states"`
}

type OrderAssignments map[string][][]bool

func elevatorSystemToHraSystem(elevSystem ElevatorSystem) hraElevatorSystem {
	hraSystem := hraElevatorSystem{
		HallRequests:   elevSystem.HallRequests,
		ElevatorStates: make(map[string]hraLocalElevatorState),
	}

	for id, state := range elevSystem.ElevatorStates {
		hraState := hraLocalElevatorState{
			Floor:       state.Floor,
			CabRequests: state.CabRequests,
		}

		switch state.Behaviour {
		case elevator.EB_Idle:
			hraState.Behaviour = "idle"
		case elevator.EB_DoorOpen:
			hraState.Behaviour = "doorOpen"
		case elevator.EB_Moving:
			hraState.Behaviour = "moving"
		}

		switch state.Direction {
		case elevio.MD_Stop:
			hraState.Direction = "stop"
		case elevio.MD_Up:
			hraState.Direction = "up"
		case elevio.MD_Down:
			hraState.Direction = "down"
		}

		hraSystem.ElevatorStates[id] = hraState
	}

	return hraSystem
}
func Encode(system ElevatorSystem) string {
	//Encode to JSON dynamically
	input, err := json.Marshal(elevatorSystemToHraSystem(system))
	if err != nil {
		fmt.Println("Error ", err)
	}
	return string(input)
}

func AssignRequests(elevatorStates string) string {
	out, err := exec.Command("./hall_request_assigner", "-i", (elevatorStates)).Output()
	if err != nil {
		fmt.Println("Error ", err)
	}
	return string(out)
}

func Decode(hraString string) OrderAssignments {
	var result OrderAssignments
	err := json.Unmarshal([]byte(hraString), &result)
	if err != nil {
		fmt.Println("Error ", err)
	}
	return result
}
