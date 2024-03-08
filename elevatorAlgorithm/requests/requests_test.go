package requests_test

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/requests"
	"elevatorAlgorithm/timer"
	"elevatorDriver/elevio"
	"testing"
	"time"
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

	testButtonEvent := elevator.Order(elevio.ButtonEvent{
		Floor:  1,
		Button: elevio.BT_HallDown,
	})
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

	//Here we should clear hall down
	testState.Floor = 3
	testState.Direction = elevio.MD_Up
	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}} //Req at floor 2, down(?)
	if !requests.ShouldClearHallDown(testState.Floor, testState.Direction, testState.Requests) {
		t.Error("Failed assert, should clear down at floor while moving down")
	}

	//Here we should not clear hall down
	testState.Floor = 2
	testState.Direction = elevio.MD_Up
	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {false, true, false}, {false, false, false}} //Req at floor 2, down(?)
	if requests.ShouldClearImmediately(testState.Floor, testState.Direction, testButtonEvent) {
		t.Error("Failed assert, should not clear down at floor while moving down")
	}

	//Here we should clear hall down
	testState.Floor = 1
	testState.Direction = elevio.MD_Up
	if !requests.ShouldClearImmediately(testState.Floor, testState.Direction, testButtonEvent) {
		t.Error("Failed assert, should clear down at floor while moving down")
	}

	//Here we should not clear hall down immediately
	testState.Floor = 1
	testState.Direction = elevio.MD_Stop
	//possible to clearImmeadiately if door is open(??)
	if !requests.ShouldClearImmediately(testState.Floor, testState.Direction, testButtonEvent) {
		t.Error("Failed assert, should clear down at floor while moving down")
	}

	testState.Requests = [][]bool{{false, true, true}, {true, false, false}, {false, true, false}, {false, false, false}} //Req at floor 2, down(?)
	testState.Floor = 3
	testState.Direction = elevio.MD_Up
	//elevator should clear call in 3rd floor hallDown
	timer.Initialize()
	timer.Start()

	if testState.Behaviour != elevator.EB_DoorOpen {
		t.Error("Failed assert, elevator door should be open")
	}

	//mu..?:D
	select {
	case <-timer.TimerChan:

	case <-time.After(5500 * time.Millisecond):
		t.Error("Failed to receive timer within expected time")
		if testState.Behaviour != elevator.EB_DoorOpen {
			t.Error("Failed assert, elevator door should be open")
		}
	}

}
