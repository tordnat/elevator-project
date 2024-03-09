package requests_test

/*
import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/requests"
	"elevatorAlgorithm/timer"
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

	//Here we should clear hall down
	testState.Floor = 3
	testState.Direction = elevio.MD_Up
	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}} //Req at floor 2, down(?)
	if !requests.ShouldClearHallDown(testState.Floor, testState.Direction, testState.Requests) {
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
	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}
	if !requests.RequestsAbove(testState.Floor, testState.Requests) {
		t.Error("Failed assert, request above")
	}

	testState.Floor = 1
	testState.Requests = [][]bool{{false, false, false}, {false, false, true}, {false, false, false}, {false, false, false}}
	if !requests.RequestsAbove() {
		t.Error("Failed assert, request above")
	}

	testState.Floor = 2
	testState.Requests = [][]bool{{true, false, true}, {false, false, true}, {true, true, true}, {false, true, true}}
	if !requests.RequestsAbove() {
		t.Error("Failed assert, request above")
	}

	testState.Floor = 4
	testState.Requests = [][]bool{{true, false, true}, {true, true, true}, {true, false, true}, {false, true, true}}
	if requests.RequestsAbove() {
		t.Error("Failed assert, no request above")
	}

	testState.Floor = 1
	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, false, false}}
	if requests.RequestsAbove() {
		t.Error("Failed assert, no request above")
	}

	//Unit tests for requests below
	testState.Floor = 4
	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}
	if !requests.RequestsAbove() {
		t.Error("Failed assert, no request below")
	}

	testState.Floor = 2
	testState.Requests = [][]bool{{false, false, false}, {false, false, true}, {false, false, false}, {false, false, false}}
	if !requests.RequestsAbove() {
		t.Error("Failed assert, no request below")
	}

	testState.Floor = 2
	testState.Requests = [][]bool{{true, false, true}, {false, false, true}, {true, true, true}, {false, true, true}}
	if requests.RequestsAbove() {
		t.Error("Failed assert, request below")
	}

	testState.Floor = 1
	testState.Requests = [][]bool{{true, false, true}, {true, true, true}, {true, false, true}, {false, true, true}}
	if !requests.RequestsAbove() {
		t.Error("Failed assert, no request below")
	}

	testState.Floor = 3
	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, false, false}}
	if !requests.RequestsAbove() {
		t.Error("Failed assert, no request below")
	}

	//Unit tests for requestsHere
	testState.Floor = 1
	testState.Requests = [][]bool{{false, false, true}, {false, false, false}, {false, false, false}, {false, false, false}}
	if !requests.RequestsAbove() {
		t.Error("Failed assert, request here")
	}

	testState.Floor = 2
	testState.Requests = [][]bool{{true, false, true}, {false, false, false}, {true, true, true}, {false, true, true}}
	if !requests.RequestsAbove() {
		t.Error("Failed assert, no request here")
	}

	testState.Floor = 4
	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {false, false, false}, {false, true, false}}
	if !requests.RequestsAbove() {
		t.Error("Failed assert, request here")
	}

	testState.Floor = 3
	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {true, true, true}, {false, false, false}}
	if !requests.RequestsAbove() {
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

	//S4
	testState.Floor = 3
	testState.Direction = elevio.MD_Up
	testState.Requests = [][]bool{{false, false, false}, {false, false, false}, {true, true, false}, {false, true, false}}
	if (testState.Behaviour != elevator.EB_DoorOpen) && (areEqual(testState.Requests, [][]bool{{false, false, false}, {false, false, false}, {true, true, false}, {false, true, false}})) {
		t.Error("Failed assert, doors should open and up order should be cleared")
	}

}

func areEqual(a, b [][]bool) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}

	return true
}
*/
