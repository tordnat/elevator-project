package requests_test

import (
	"elevatorControl/elevator"
	"elevatorControl/fsm"
	"elevatorControl/requests"
	"elevatorControl/timer"
	"elevatorDriver/elevio"
	"testing"
)

func TestOrders(t *testing.T) {
	testState := elevator.Elevator{
		Behaviour: elevator.EB_Moving,
		Floor:     1,
		Direction: elevio.MD_Up,
	}

	//Here we should clear hall down
	testShouldClearHallDown := requests.ShouldClearHallDown(testState.Floor, testState.Direction, testState.Orders)
	if testShouldClearHallDown {
		t.Error("Failed assert, should not clear down at floor while moving up")
	}

	//Here we should not clear hall down
	testState.Direction = elevio.MD_Down
	testShouldNotClearHallDown := requests.ShouldClearHallDown(testState.Floor, testState.Direction, testState.Orders)
	if !testShouldNotClearHallDown {
		t.Error("Failed assert, should clear down at floor while moving down")
	}

	//Here we should clear hall down
	testState.Floor = 3
	testState.Direction = elevio.MD_Up
	orders := [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}} //Req at floor 2, down(?)
	if !requests.ShouldClearHallDown(testState.Floor, testState.Direction, orders) {
		t.Error("Failed assert, should clear down at floor while moving up")
	}

	//elevator should not clear hall up
	testState.Orders = [][]bool{{false, true, true}, {true, false, false}, {false, true, false}, {true, false, false}} //Req at floor 2, down(?)
	testState.Floor = 3
	testState.Direction = elevio.MD_Down
	if requests.ShouldClearHallUp(testState.Floor, testState.Direction, testState.Orders) {
		t.Error("Failed assert, should not clear hall up")
	}

	//elevator should clear hall up
	testState.Direction = elevio.MD_Up
	if !requests.ShouldClearHallUp(testState.Floor, testState.Direction, testState.Orders) {
		t.Error("Failed assert, should clear hall up")
	}

	//Unit tests for requests above
	testState.Floor = 2
	testState.Orders = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}
	if !requests.OrdersAbove(testState.Floor, testState.Orders) {
		t.Error("Failed assert, request above")
	}

	testState.Floor = 0
	testState.Orders = [][]bool{{false, false, false}, {false, false, true}, {false, false, false}, {false, false, false}}
	if !requests.OrdersAbove(testState.Floor, testState.Orders) {
		t.Error("Failed assert, request above")
	}

	testState.Floor = 1
	testState.Orders = [][]bool{{true, false, true}, {false, false, true}, {true, true, true}, {false, true, true}}
	if !requests.OrdersAbove(testState.Floor, testState.Orders) {
		t.Error("Failed assert, request above")
	}

	testState.Floor = 3
	testState.Orders = [][]bool{{true, false, true}, {true, true, true}, {true, false, true}, {false, true, true}}
	if requests.OrdersAbove(testState.Floor, testState.Orders) {
		t.Error("Failed assert, no request above")
	}

	testState.Floor = 0
	testState.Orders = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, false, false}}
	if requests.OrdersAbove(testState.Floor, testState.Orders) {
		t.Error("Failed assert, no request above")
	}

	//Unit tests for requests below
	testState.Floor = 3
	testState.Orders = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}
	if !requests.OrdersBelow(testState.Floor, testState.Orders) {
		t.Error("Failed assert, no request below")
	}

	testState.Floor = 1
	testState.Orders = [][]bool{{false, false, false}, {false, false, true}, {false, false, false}, {false, false, false}}
	if !requests.OrdersBelow(testState.Floor, testState.Orders) {
		t.Error("Failed assert, no request below")
	}

	testState.Floor = 1
	testState.Orders = [][]bool{{true, false, true}, {false, false, true}, {true, true, true}, {false, true, true}}
	if requests.OrdersBelow(testState.Floor, testState.Orders) {
		t.Error("Failed assert, request below")
	}

	testState.Floor = 0
	testState.Orders = [][]bool{{true, false, true}, {true, true, true}, {true, false, true}, {false, true, true}}
	if !requests.OrdersBelow(testState.Floor, testState.Orders) {
		t.Error("Failed assert, no request below")
	}

	testState.Floor = 2
	testState.Orders = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, false, false}}
	if !requests.OrdersBelow(testState.Floor, testState.Orders) {
		t.Error("Failed assert, no request below")
	}

	//Unit tests for requestsHere
	testState.Floor = 0
	testState.Orders = [][]bool{{false, false, true}, {false, false, false}, {false, false, false}, {false, false, false}}
	if !requests.OrdersHere(testState.Floor, testState.Orders) {
		t.Error("Failed assert, request here")
	}

	testState.Floor = 1
	testState.Orders = [][]bool{{true, false, true}, {false, false, false}, {true, true, true}, {false, true, true}}
	if !requests.OrdersHere(testState.Floor, testState.Orders) {
		t.Error("Failed assert, no request here")
	}

	testState.Floor = 3
	testState.Orders = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}
	if !requests.OrdersHere(testState.Floor, testState.Orders) {
		t.Error("Failed assert, request here")
	}

	testState.Floor = 2
	testState.Orders = [][]bool{{false, false, false}, {false, false, false}, {true, true, true}, {false, false, false}}
	if !requests.OrdersHere(testState.Floor, testState.Orders) {
		t.Error("Failed assert, no request here")
	}

	//Unit tests for choose direction

	testState.Floor = 2
	testState.Orders = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}

	tmpBehaviourPair := requests.DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
	if requests.ChooseDirection(elevio.MD_Up, testState.Floor, testState.Orders) == tmpBehaviourPair {
		t.Error("Failsed assert, dirnBehaviourPair doesnt match")
	}

	tmpBehaviourPair = requests.DirnBehaviourPair{elevio.MD_Up, elevator.EB_DoorOpen}
	if requests.ChooseDirection(elevio.MD_Down, 3, [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}) == tmpBehaviourPair {
		t.Error("Failed assert, dirnBehaviourPair doesnt match")
	}

	tmpBehaviourPair = requests.DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
	if requests.ChooseDirection(elevio.MD_Stop, 2, [][]bool{{false, true, false}, {false, false, false}, {false, false, false}, {false, false, false}}) == tmpBehaviourPair {
		t.Error("Failed assert, dirnBehaviourPair doesnt match")
	}

	//S4
	testState.Floor = 2
	testState.Direction = elevio.MD_Up
	testState.Orders = [][]bool{{false, false, true}, {false, false, false}, {true, true, false}, {false, false, false}}
	doorTimer, obstructionTimer, inactivityTimer := timer.InitTimers()
	testState.Behaviour, _ = fsm.OnFloorArrival(testState, doorTimer, inactivityTimer, false, obstructionTimer)

	if testState.Direction != elevio.MD_Up {
		t.Error("Failed assert, MD should be up")
	}
	if testState.Behaviour != elevator.EB_DoorOpen {
		t.Error("Failed assert, door should be open")
	}

	//Clearing order manually
	testState.Orders = [][]bool{{false, false, true}, {false, false, false}, {false, true, false}, {false, false, false}}
	requests.ChooseDirection(testState.Direction, testState.Floor, testState.Orders)
	if testState.Direction != elevio.MD_Down {
		t.Error("Failed assert, MD should be down")
	}
	if testState.Behaviour != elevator.EB_DoorOpen {
		t.Error("Failed assert, door should be open")
	}

	testState.Orders = [][]bool{{false, false, true}, {false, false, false}, {false, false, false}, {false, false, false}}
	requests.ChooseDirection(testState.Direction, testState.Floor, testState.Orders)
	if testState.Direction != elevio.MD_Down {
		t.Error("Failed assert, MD should be down")
	}
	if testState.Behaviour != elevator.EB_Moving {
		t.Error("Failed assert, EB should be moving")
	}

	testState.Orders = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, false, false}}
	if requests.HaveOrders(testState.Floor, testState.Orders) {
		t.Error("Failed assert, no orders")
	}
	testState.Orders = [][]bool{{false, true, false}, {false, false, false}, {false, false, false}, {false, false, false}}
	if !requests.HaveOrders(testState.Floor, testState.Orders) {
		t.Error("Failed assert, orders")
	}

}
