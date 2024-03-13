package orders_test

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/fsm"
	"elevatorAlgorithm/orders"
	"elevatorAlgorithm/timer"
	"elevatorDriver/elevio"
	"testing"
)

func TestOrders(t *testing.T) {
	testState := elevator.Elevator{
		Behaviour: elevator.EB_Moving,
		Floor:     1,
		Direction: elevio.MD_Up,
		Orders: [][]bool{
			{true, true, true},
			{true, true, false},
			{false, false, false},
			{false, false, false}},
	}

	//Here we should clear hall down
	testShouldClearHallDown := orders.ShouldClearHallDown(testState.Floor, testState.Direction, testState.Orders)
	if testShouldClearHallDown {
		t.Error("Failed assert, should not clear down at floor while moving up")
	}

	//Here we should not clear hall down
	testState.Direction = elevio.MD_Down
	testShouldNotClearHallDown := orders.ShouldClearHallDown(testState.Floor, testState.Direction, testState.Orders)
	if !testShouldNotClearHallDown {
		t.Error("Failed assert, should clear down at floor while moving down")
	}

	//Here we should clear hall down
	testState.Floor = 3
	testState.Direction = elevio.MD_Up
	testState.Orders = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}} //Req at floor 2, down(?)
	if !orders.ShouldClearHallDown(testState.Floor, testState.Direction, testState.Orders) {
		t.Error("Failed assert, should clear down at floor while moving up")
	}

	//elevator should not clear hall up
	testState.Orders = [][]bool{{false, true, true}, {true, false, false}, {false, true, false}, {true, false, false}} //Req at floor 2, down(?)
	testState.Floor = 3
	testState.Direction = elevio.MD_Down
	if orders.ShouldClearHallUp(testState.Floor, testState.Direction, testState.Orders) {
		t.Error("Failed assert, should not clear hall up")
	}

	//elevator should clear hall up
	testState.Direction = elevio.MD_Up
	if !orders.ShouldClearHallUp(testState.Floor, testState.Direction, testState.Orders) {
		t.Error("Failed assert, should clear hall up")
	}

	//Unit tests for orders above
	testState.Floor = 2
	testState.Orders = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}
	if !orders.OrdersAbove(testState.Floor, testState.Orders) {
		t.Error("Failed assert, order above")
	}

	testState.Floor = 0
	testState.Orders = [][]bool{{false, false, false}, {false, false, true}, {false, false, false}, {false, false, false}}
	if !orders.OrdersAbove(testState.Floor, testState.Orders) {
		t.Error("Failed assert, order above")
	}

	testState.Floor = 1
	testState.Orders = [][]bool{{true, false, true}, {false, false, true}, {true, true, true}, {false, true, true}}
	if !orders.OrdersAbove(testState.Floor, testState.Orders) {
		t.Error("Failed assert, order above")
	}

	testState.Floor = 3
	testState.Orders = [][]bool{{true, false, true}, {true, true, true}, {true, false, true}, {false, true, true}}
	if orders.OrdersAbove(testState.Floor, testState.Orders) {
		t.Error("Failed assert, no order above")
	}

	testState.Floor = 0
	testState.Orders = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, false, false}}
	if orders.OrdersAbove(testState.Floor, testState.Orders) {
		t.Error("Failed assert, no order above")
	}

	//Unit tests for orders below
	testState.Floor = 3
	testState.Orders = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}
	if orders.OrdersBelow(testState.Floor, testState.Orders) {
		t.Error("Failed assert, no order below")
	}

	testState.Floor = 1
	testState.Orders = [][]bool{{false, false, false}, {false, false, true}, {false, false, false}, {false, false, false}}
	if orders.OrdersBelow(testState.Floor, testState.Orders) {
		t.Error("Failed assert, no order below")
	}

	testState.Floor = 1
	testState.Orders = [][]bool{{true, false, true}, {false, false, true}, {true, true, true}, {false, true, true}}
	if !orders.OrdersBelow(testState.Floor, testState.Orders) {
		t.Error("Failed assert, order below")
	}

	testState.Floor = 0
	testState.Orders = [][]bool{{true, false, true}, {true, true, true}, {true, false, true}, {false, true, true}}
	if orders.OrdersBelow(testState.Floor, testState.Orders) {
		t.Error("Failed assert, no order below")
	}

	testState.Floor = 2
	testState.Orders = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, false, false}}
	if orders.OrdersBelow(testState.Floor, testState.Orders) {
		t.Error("Failed assert, no order below")
	}

	//Unit tests for ordersHere
	testState.Floor = 0
	testState.Orders = [][]bool{{false, false, true}, {false, false, false}, {false, false, false}, {false, false, false}}
	if !orders.OrdersHere(testState.Floor, testState.Orders) {
		t.Error("Failed assert, order here")
	}

	testState.Floor = 1
	testState.Orders = [][]bool{{true, false, true}, {false, false, false}, {true, true, true}, {false, true, true}}
	if orders.OrdersHere(testState.Floor, testState.Orders) {
		t.Error("Failed assert, no order here")
	}

	testState.Floor = 3
	testState.Orders = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}
	if !orders.OrdersHere(testState.Floor, testState.Orders) {
		t.Error("Failed assert, order here")
	}

	testState.Floor = 2
	testState.Orders = [][]bool{{false, false, false}, {false, false, false}, {true, true, true}, {false, false, false}}
	if !orders.OrdersHere(testState.Floor, testState.Orders) {
		t.Error("Failed assert, no order here")
	}

	//Unit tests for choose direction

	testState.Floor = 2
	testState.Orders = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}

	tmpBehaviourPair := orders.DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
	if !orders.ChooseDirection(elevio.MD_Up, testState.Floor, testState.Orders) == tmpBehaviourPair {
		t.Error("Failsed assert, dirnBehaviourPair doesnt match")
	}

	tmpBehaviourPair = orders.DirnBehaviourPair{elevio.MD_Up, elevator.EB_DoorOpen}
	if !orders.ChooseDirection(elevio.MD_Down, 3, [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}) == tmpBehaviourPair {
		t.Error("Failed assert, dirnBehaviourPair doesnt match")
	}

	tmpBehaviourPair = orders.DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
	if !orders.ChooseDirection(elevio.MD_Stop, 2, [][]bool{{false, true, false}, {false, false, false}, {false, false, false}, {false, false, false}}) == tmpBehaviourPair {
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
	orders.ChooseDirection(testState.Direction, testState.Floor, testState.Orders)
	if testState.Direction != elevio.MD_Down {
		t.Error("Failed assert, MD should be down")
	}
	if testState.Behaviour != elevator.EB_DoorOpen {
		t.Error("Failed assert, door should be open")
	}

	testState.Orders = [][]bool{{false, false, true}, {false, false, false}, {false, false, false}, {false, false, false}}
	orders.ChooseDirection(testState.Direction, testState.Floor, testState.Orders)
	if testState.Direction != elevio.MD_Down {
		t.Error("Failed assert, MD should be down")
	}
	if testState.Behaviour != elevator.EB_Moving {
		t.Error("Failed assert, EB should be moving")
	}

	testState.Orders = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, false, false}}
	if orders.HaveOrders(testState.Floor, testState.Orders) {
		t.Error("Failed assert, no orders")
	}
	testState.Orders = [][]bool{{false, true, false}, {false, false, false}, {false, false, false}, {false, false, false}}
	if !orders.HaveOrders(testState.Floor, testState.Orders) {
		t.Error("Failed assert, orders")
	}

}
