package requests_test

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/fsm"
	"elevatorAlgorithm/requests"
	"elevatorAlgorithm/timer"
	"elevatorDriver/elevio"
	"testing"
)

func TestRequests(t *testing.T) {
	testState := elevator.Elevator{
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

	//Here we should clear hall down
	testState.Floor = 3
	testState.Direction = elevio.MD_Up
	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}} //Req at floor 2, down(?)
	if !requests.ShouldClearHallDown(testState.Floor, testState.Direction, testState.Requests) {
		t.Error("Failed assert, should clear down at floor while moving down")
	}

	//elevator should not clear hall up
	testState.Requests = [][]bool{{false, true, true}, {true, false, false}, {false, true, false}, {false, false, false}} //Req at floor 2, down(?)
	testState.Floor = 3
	testState.Direction = elevio.MD_Down
	if !requests.ShouldClearHallUp(testState.Floor, testState.Direction, testState.Requests) {
		t.Error("Failed assert, elevator door should be open")
	}

	//elevator should clear hall up
	testState.Direction = elevio.MD_Up
	if requests.ShouldClearHallUp(testState.Floor, testState.Direction, testState.Requests) {
		t.Error("Failed assert, elevator door should be open")
	}

	//Unit tests for requests above
	testState.Floor = 3
	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}
	if !requests.RequestsAbove(testState.Floor, testState.Requests) {
		t.Error("Failed assert, request above")
	}

	testState.Floor = 1
	testState.Requests = [][]bool{{false, false, false}, {false, false, true}, {false, false, false}, {false, false, false}}
	if !requests.RequestsAbove(testState.Floor, testState.Requests) {
		t.Error("Failed assert, request above")
	}

	testState.Floor = 2
	testState.Requests = [][]bool{{true, false, true}, {false, false, true}, {true, true, true}, {false, true, true}}
	if !requests.RequestsAbove(testState.Floor, testState.Requests) {
		t.Error("Failed assert, request above")
	}

	testState.Floor = 4
	testState.Requests = [][]bool{{true, false, true}, {true, true, true}, {true, false, true}, {false, true, true}}
	if requests.RequestsAbove(testState.Floor, testState.Requests) {
		t.Error("Failed assert, no request above")
	}

	testState.Floor = 1
	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, false, false}}
	if requests.RequestsAbove(testState.Floor, testState.Requests) {
		t.Error("Failed assert, no request above")
	}

	//Unit tests for requests below
	testState.Floor = 4
	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}
	if !requests.RequestsBelow(testState.Floor, testState.Requests) {
		t.Error("Failed assert, no request below")
	}

	testState.Floor = 2
	testState.Requests = [][]bool{{false, false, false}, {false, false, true}, {false, false, false}, {false, false, false}}
	if !requests.RequestsBelow(testState.Floor, testState.Requests) {
		t.Error("Failed assert, no request below")
	}

	testState.Floor = 2
	testState.Requests = [][]bool{{true, false, true}, {false, false, true}, {true, true, true}, {false, true, true}}
	if requests.RequestsBelow(testState.Floor, testState.Requests) {
		t.Error("Failed assert, request below")
	}

	testState.Floor = 1
	testState.Requests = [][]bool{{true, false, true}, {true, true, true}, {true, false, true}, {false, true, true}}
	if !requests.RequestsBelow(testState.Floor, testState.Requests) {
		t.Error("Failed assert, no request below")
	}

	testState.Floor = 3
	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, false, false}}
	if !requests.RequestsBelow(testState.Floor, testState.Requests) {
		t.Error("Failed assert, no request below")
	}

	//Unit tests for requestsHere
	testState.Floor = 1
	testState.Requests = [][]bool{{false, false, true}, {false, false, false}, {false, false, false}, {false, false, false}}
	if !requests.RequestsHere(testState.Floor, testState.Requests) {
		t.Error("Failed assert, request here")
	}

	testState.Floor = 2
	testState.Requests = [][]bool{{true, false, true}, {false, false, false}, {true, true, true}, {false, true, true}}
	if !requests.RequestsHere(testState.Floor, testState.Requests) {
		t.Error("Failed assert, no request here")
	}

	testState.Floor = 4
	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}
	if !requests.RequestsHere(testState.Floor, testState.Requests) {
		t.Error("Failed assert, request here")
	}

	testState.Floor = 3
	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {true, true, true}, {false, false, false}}
	if !requests.RequestsHere(testState.Floor, testState.Requests) {
		t.Error("Failed assert, no request here")
	}

	//Unit tests for choose direction

	testState.Floor = 1
	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}

	tmpBehaviourPair := requests.DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
	if requests.ChooseDirection(elevio.MD_Up, testState.Floor, testState.Requests) == tmpBehaviourPair {
		t.Error("Failsed assert, dirnBehaviourPair doesnt match")
	}

	tmpBehaviourPair = requests.DirnBehaviourPair{elevio.MD_Up, elevator.EB_DoorOpen}
	if requests.ChooseDirection(elevio.MD_Down, 4, [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}) == tmpBehaviourPair {
		t.Error("Failed assert, dirnBehaviourPair doesnt match")
	}

	tmpBehaviourPair = requests.DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
	if requests.ChooseDirection(elevio.MD_Stop, 3, [][]bool{{false, true, false}, {false, false, false}, {false, false, false}, {false, false, false}}) == tmpBehaviourPair {
		t.Error("Failed assert, dirnBehaviourPair doesnt match")
	}

	tmpBehaviourPair = requests.DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
	if requests.ChooseDirection(elevio.MD_Up, 1, [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, false, false}}) == tmpBehaviourPair {
		t.Error("Failed assert, dirnBehaviourPair doesnt match")
	}
	//Floors might be shifted by 1

	//S4
	testState.Floor = 2
	testState.Direction = elevio.MD_Up
	testState.Requests = [][]bool{{false, false, true}, {false, false, false}, {true, true, false}, {false, false, false}}
	doorTimer, obstructionTimer, inactivityTimer := timer.InitTimers()
	testState.Behaviour, _ = fsm.OnFloorArrival(testState, doorTimer, inactivityTimer, false, obstructionTimer)

	if testState.Direction != elevio.MD_Up {
		t.Error("Failed assert, MD should be up")
	}
	if testState.Behaviour != elevator.EB_DoorOpen {
		t.Error("Failed assert, door should be open")
	}

	//Clearing order manually
	testState.Requests = [][]bool{{false, false, true}, {false, false, false}, {false, true, false}, {false, false, false}}
	requests.ChooseDirection(testState.Direction, testState.Floor, testState.Requests)
	if testState.Direction != elevio.MD_Down {
		t.Error("Failed assert, MD should be down")
	}
	if testState.Behaviour != elevator.EB_DoorOpen {
		t.Error("Failed assert, door should be open")
	}

	testState.Requests = [][]bool{{false, false, true}, {false, false, false}, {false, false, false}, {false, false, false}}
	requests.ChooseDirection(testState.Direction, testState.Floor, testState.Requests)
	if testState.Direction != elevio.MD_Down {
		t.Error("Failed assert, MD should be down")
	}
	if testState.Behaviour != elevator.EB_Moving {
		t.Error("Failed assert, EB should be moving")
	}

	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, false, false}}
	if requests.HaveOrders(testState.Floor, testState.Requests) {
		t.Error("Failed assert, no orders")
	}
	testState.Requests = [][]bool{{false, true, false}, {false, false, false}, {false, false, false}, {false, false, false}}
	if !requests.HaveOrders(testState.Floor, testState.Requests) {
		t.Error("Failed assert, orders")
	}

}
