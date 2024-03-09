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
	if requests.ClearAtFloor(testState.Floor, testState.Direction, testButtonEvent) {
		t.Error("Failed assert, should not clear down at floor while moving down")
	}

	//Here we should clear hall down
	testState.Floor = 1
	testState.Direction = elevio.MD_Up
	if !requests.ClearAtFloor(testState.Floor, testState.Direction, testButtonEvent) {
		t.Error("Failed assert, should clear down at floor while moving down")
	}

	//Here we should not clear hall down immediately
	testState.Floor = 1
	testState.Direction = elevio.MD_Stop
	//possible to clearImmeadiately if door is open(??)
	if !requests.ClearAtFloor(testState.Floor, testState.Direction, testButtonEvent) {
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


	//Unit tests for requests above
	testState.Floor = 3
	testState.Requests = {{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}
	if !requests.requestsAbove(){
		t.Error("Failes assert, request above")
	}

	testState.Floor = 1
	testState.Requests = {{false, false, false}, {false, false, true}, {false, false, false}, {false, false, false}}
	if !requests.requestsAbove(){
		t.Error("Failes assert, request above")
	}

	testState.Floor = 2
	testState.Requests = {{true, false, true}, {false, false, true}, {true, true, true}, {false, true, true}}
	if !requests.requestsAbove(){
		t.Error("Failes assert, request above")
	}

	testState.Floor = 4
	testState.Requests = {{true, false, true}, {true, true, true}, {true, false, true}, {false, true, true}}
	if requests.requestsAbove(){
		t.Error("Failes assert, no request above")
	}

	testState.Floor = 1
	testState.Requests = {{false, false, false}, {false, false, false}, {false, false, false}, {false, false, false}}
	if requests.requestsAbove(){
		t.Error("Failes assert, no request above")
	}

	
	//Unit tests for requests below
	testState.Floor = 4
	testState.Requests = {{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}
	if !requests.requestsAbove(){
		t.Error("Failes assert, no request below")
	}

	testState.Floor = 2
	testState.Requests = {{false, false, false}, {false, false, true}, {false, false, false}, {false, false, false}}
	if !requests.requestsAbove(){
		t.Error("Failes assert, no request below")
	}

	testState.Floor = 2
	testState.Requests = {{true, false, true}, {false, false, true}, {true, true, true}, {false, true, true}}
	if requests.requestsAbove(){
		t.Error("Failes assert, request below")
	}

	testState.Floor = 1
	testState.Requests = {{true, false, true}, {true, true, true}, {true, false, true}, {false, true, true}}
	if !requests.requestsAbove(){
		t.Error("Failes assert, no request below")
	}

	testState.Floor = 3
	testState.Requests = {{false, false, false}, {false, false, false}, {false, false, false}, {false, false, false}}
	if !requests.requestsAbove(){
		t.Error("Failes assert, no request below")
	}


	//Unit tests for requestsHere
	testState.Floor = 1
	testState.Requests = {{false, false, true}, {false, false false}, {false, false, false}, {false, false, false}}
	if !requests.requestsAbove(){
		t.Error("Failes assert, request here")
	}

	testState.Floor = 2
	testState.Requests = {{true, false, true}, {false, false, false}, {true, true, true}, {false, true, true}}
	if !requests.requestsAbove(){
		t.Error("Failes assert, no request here")
	}

	testState.Floor = 4
	testState.Requests = {{false, false false}, {false, false false}, {false, false false}, {false, true, false}}
	if !requests.requestsAbove(){
		t.Error("Failes assert, request here")
	}

	testState.Floor = 3
	testState.Requests = {{false, false, false}, {false, false, false}, {true, true, true}, {false, false, false}}
	if !requests.requestsAbove(){
		t.Error("Failes assert, no request here")
	}


	//Unit tests for choose direction

	testState.Floor = 1
	testState.Requests = {{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}
	if requests.ChooseDirection(elevio.MD_Up, testState.Floor, testState.Requests) == requests.DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving} {
		t.Error("Failes assert, direction should be up")
	}

	if requests.ChooseDirection(elevio.MD_Up, 3, {{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}) == requests.DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving} {
		t.Error("Failes assert, direction should be up")
	}

}
