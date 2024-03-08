package requests_test

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/requests"
	"elevatorDriver/elevio"
	"testing"
)

func TestRequests(t *testing.T) {
	testState := elevator.ElevatorState{
		Behaviour: elevator.EB_Moving,
		Floor:     1,
		Direction: elevio.MD_Up,
		Requests: [][]bool{
			{true, true, true},
			{true, true, false},
			{false, false, false},
			{false, false, false}},
	}
	//Here we should clear hall down
	testShouldClearHallDown := requests.ShouldClearHallDown(testState.Floor, testState.Direction, testState.Requests)
	if testShouldClearHallDown {
		t.Error("Failed assert, should not clear down at floor while moving up")
	}

	//Here we should not clear hall down
	testState.Direction = elevio.MD_Down
	testShouldNotClearHallDown := requests.ShouldClearHallDown(testState.Floor, testState.Direction, testState.Requests)
	if !testShouldNotClearHallDown {
		t.Error("Failed assert, should clear down at floor while moving down")
	}
}
