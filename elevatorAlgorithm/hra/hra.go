package hra

import (
	"elevatorAlgorithm/elevator"
	"elevatorDriver/elevio"
	"encoding/json"
	"fmt"

	//"encoding/json"
	"os/exec"
)

const (
	unknownOrder     = 0
	noOrder          = 1
	unconfirmedOrder = 2
	confirmedOrder   = 3
)

// TODO: Move this to a different file
type LocalElevatorState struct {
	Behaviour elevator.ElevatorBehaviour
	Floor     int
	Direction elevio.MotorDirection
	CabOrders []int
}

type HallOrdersType [][]int

type ElevatorSystem struct {
	HallOrders [][]int

	ElevatorStates map[string]LocalElevatorState
}

type hraLocalElevatorState struct {
	Behaviour string `json:"behaviour"` // "idle", "moving", or "doorOpen"
	Floor     int    `json:"floor"`
	Direction string `json:"direction"` // "up", "down", or "stop"
	CabOrders []bool `json:"cabRequests"`
}

type hraElevatorSystem struct {
	HallOrders     [][]bool                         `json:"hallRequests"`
	ElevatorStates map[string]hraLocalElevatorState `json:"states"`
}

type OrderAssignments map[string][][]bool

func elevatorSystemToHraSystem(elevSystem ElevatorSystem) hraElevatorSystem {
	hraSystem := hraElevatorSystem{
		HallOrders:     hraHallorderTypeToBool(elevSystem.HallOrders),
		ElevatorStates: make(map[string]hraLocalElevatorState),
	}

	for id, state := range elevSystem.ElevatorStates {
		hraState := hraLocalElevatorState{
			Floor:     state.Floor,
			CabOrders: hraCaborderTypeToBool(state.CabOrders),
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
	input, err := json.Marshal(elevatorSystemToHraSystem(system))
	if err != nil {
		fmt.Println("Error ", err)
	}
	return string(input)
}

func AssignOrders(elevatorStates string) string {
	out, err := exec.Command("./hall_request_assigner", "--includeCab", "-i", (elevatorStates)).Output()
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

func hraHallorderTypeToBool(orders [][]int) [][]bool {
	retArr := make([][]bool, len(orders))
	for i, row := range orders {
		retArr[i] = make([]bool, len(row))
		for j, req := range row {
			if req == confirmedOrder {
				retArr[i][j] = true
			} else {
				retArr[i][j] = false
			}
		}
	}
	return retArr
}

func hraCaborderTypeToBool(orders []int) []bool {
	retArr := make([]bool, len(orders))
	for i, req := range orders {
		if req == confirmedOrder {
			retArr[i] = true
		} else {
			retArr[i] = false
		}
	}
	return retArr
}
