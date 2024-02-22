package main

import (
	"encoding/json"
	"fmt"

	//"encoding/json"
	"os/exec"
)

type State struct {
	Behaviour   string `json:"behaviour"` // "idle", "moving", or "doorOpen"
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"` // "up", "down", or "stop"
	CabRequests []bool `json:"cabRequests"`
}

type ElevatorSystem struct {
	HallRequests [][]bool `json:"hallRequests"`
	States       []State  `json:"states"`
}
type OrderAssignments map[string][][]bool

func (e *ElevatorSystem) toMap() map[string]State {
	statesMap := make(map[string]State)
	for i, state := range e.States {
		key := fmt.Sprintf("%d", i)
		statesMap[key] = state
	}
	return statesMap
}

//Example usage. Remove when module is incorporated
/*
func main() {
	system := ElevatorSystem{
		HallRequests: [][]bool{
			{false, false}, {true, false}, {false, false}, {false, true},
		},
		States: []State{
			{
				Behaviour:   "moving",
				Floor:       2,
				Direction:   "up",
				CabRequests: []bool{false, false, true, true},
			},
			{
				Behaviour:   "idle",
				Floor:       0,
				Direction:   "stop",
				CabRequests: []bool{false, false, false, false},
			},
		},
	}
	input := encode(system)

	fmt.Println((input))

	hraString := HRA(input)
	fmt.Println(hraString)
	orders := decode(hraString)
	fmt.Printf("Result: %+v\n", orders["1"])
}
*/
func encode(system ElevatorSystem) string {
	//Encode to JSON dynamically
	statesMap := system.toMap()
	input, err := json.Marshal(struct {
		HallRequests [][]bool         `json:"hallRequests"`
		States       map[string]State `json:"states"`
	}{
		HallRequests: system.HallRequests,
		States:       statesMap,
	})
	if err != nil {
		fmt.Println("Error ", err)
	}
	return string(input)
}

func HRA(elevatorStates string) string {
	out, err := exec.Command("./hall_request_assigner", "-i", (elevatorStates)).Output()
	if err != nil {
		fmt.Println("Error ", err)
	}
	return string(out)
}

func decode(hraString string) OrderAssignments {
	var result OrderAssignments
	err := json.Unmarshal([]byte(hraString), &result)
	if err != nil {
		fmt.Println("Error ", err)
	}
	return result
}
