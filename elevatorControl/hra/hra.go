package hra

import (
	"elevatorControl/elevator"
	"encoding/json"
	"fmt"
	"os/exec"
)

type HallOrdersType [][]int

type HraLocalElevatorState struct {
	Behaviour string `json:"behaviour"` // "idle", "moving", or "doorOpen"
	Floor     int    `json:"floor"`
	Direction string `json:"direction"` // "up", "down", or "stop"
	CabOrders []bool `json:"cabRequests"`
}

type HraElevatorSystem struct {
	HallOrders     [][]bool                         `json:"hallRequests"`
	ElevatorStates map[string]HraLocalElevatorState `json:"states"`
}

type OrderAssignments map[string][][]bool

func Encode(system HraElevatorSystem) string {
	input, err := json.Marshal(system)
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

func NewElevatorSystem(floors int) HraElevatorSystem {
	hraElevSys := HraElevatorSystem{}
	hraElevSys.HallOrders = make([][]bool, floors)
	for i := 0; i < floors; i++ {
		hraElevSys.HallOrders[i] = make([]bool, elevator.N_HALL_BUTTONS)
	}
	hraElevSys.ElevatorStates = make(map[string]HraLocalElevatorState)
	return hraElevSys
}
