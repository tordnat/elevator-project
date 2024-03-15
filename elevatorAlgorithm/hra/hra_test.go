package hra_test

import (
	"elevatorAlgorithm/hra"
	"fmt"
	"testing"
)

func TestHra(t *testing.T) {
	system := hra.HraElevatorSystem{
		HallOrders: [][]bool{
			{false, false}, {true, false}, {false, false}, {false, true},
		},
		ElevatorStates: map[string]hra.HraLocalElevatorState{
			"0": {
				Behaviour: "moving",
				Floor:     2,
				Direction: "up",
				CabOrders: []bool{false, false, true, true},
			},
			"1": {
				Behaviour: "idle",
				Floor:     0,
				Direction: "stop",
				CabOrders: []bool{false, false, false, false},
			},
		},
	}
	input := hra.Encode(system)

	hraString := hra.AssignOrders(input)
	orders := hra.Decode(hraString)
	orderString := fmt.Sprintf("%+v", orders["1"])
	if orderString != "[[false false false] [true false false] [false false false] [false false false]]" {
		t.Error("Failed assert, output not equal to test case")
	}
}
