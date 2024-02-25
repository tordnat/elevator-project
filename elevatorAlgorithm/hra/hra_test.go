package hra_test

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/hra"
	"elevatorDriver/elevio"
	"fmt"
	"testing"
)

// Example usage. Remove when module is incorporated
func TestHra(t *testing.T) {
	system := hra.ElevatorSystem{
		HallRequests: [][]bool{
			{false, false}, {true, false}, {false, false}, {false, true},
		},
		ElevatorStates: map[string]hra.LocalElevatorState{
			"0": {
				Behaviour:   elevator.EB_Moving,
				Floor:       2,
				Direction:   elevio.MD_Up,
				CabRequests: []bool{false, false, true, true},
			},
			"1": {
				Behaviour:   elevator.EB_Idle,
				Floor:       0,
				Direction:   elevio.MD_Stop,
				CabRequests: []bool{false, false, false, false},
			},
		},
	}
	input := hra.Encode(system)

	hraString := hra.AssignRequests(input)
	orders := hra.Decode(hraString)
	orderString := fmt.Sprintf("%+v", orders["1"])
	if orderString != "[[false false false] [true false false] [false false false] [false false false]]" {
		t.Error("Failed assert, output not equal to test case")
	}
}
