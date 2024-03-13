package hra_test

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/hra"
	"elevatorDriver/elevio"
	"fmt"
	"testing"
)

const (
	unknownOrder     = 0
	noOrder          = 1
	unconfirmedOrder = 2
	confirmedOrder   = 3
)

// Example usage. Remove when module is incorporated
func TestHra(t *testing.T) {
	system := hra.ElevatorSystem{
		HallOrders: [][]int{
			{noOrder, noOrder}, {confirmedOrder, noOrder}, {noOrder, noOrder}, {noOrder, confirmedOrder},
		},
		ElevatorStates: map[string]hra.LocalElevatorState{
			"0": {
				Behaviour: elevator.EB_Moving,
				Floor:     2,
				Direction: elevio.MD_Up,
				CabOrders: []int{noOrder, noOrder, confirmedOrder, confirmedOrder},
			},
			"1": {
				Behaviour: elevator.EB_Idle,
				Floor:     0,
				Direction: elevio.MD_Stop,
				CabOrders: []int{noOrder, noOrder, noOrder, noOrder},
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
