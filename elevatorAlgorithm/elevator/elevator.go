package elevator

import (
	"elevatorDriver/elevio"
)

const N_FLOORS = 4
const N_BUTTONS = 3
const N_HALL_BUTTONS = 2
const DOOR_OPEN_DURATION_S = 2

type ElevatorBehaviour int

// This should be global or imported
const (
	unknownOrder = iota
	noOrder
	unconfirmedOrder
	confirmedOrder
	servicedOrder
)

const (
	EB_Idle ElevatorBehaviour = iota
	EB_DoorOpen
	EB_Moving
)

type Elevator struct {
	Behaviour ElevatorBehaviour
	Floor     int
	Direction elevio.MotorDirection
	Requests  [][]bool //This should be filled from HRA
}

type Order elevio.ButtonEvent

func NewElevator() Elevator {
	return Elevator{
		Floor:     -1,
		Direction: elevio.MD_Stop,
		Behaviour: EB_Idle,
	}
}

func (eb ElevatorBehaviour) String() string {
	if eb == EB_Idle {
		return "EB_Idle"
	} else if eb == EB_DoorOpen {
		return "EB_DoorOpen"
	} else if eb == EB_Moving {
		return "EB_Moving"
	} else {
		return "EB_UNDEFINED"
	}
}

/* Commented because it imports from HRA which creates an import cycle. Type ElevatorSystem should not be in hra, so can be uncommented when that is removed
func PrintElevatorSystem(elevatorSystem hra.ElevatorSystem) {
	fmt.Println("Elevator System State:")
	for id, state := range elevatorSystem.ElevatorStates {
		fmt.Printf("Elevator ID: %s\n", id)
		fmt.Printf("  Behaviour: %s, Floor: %d, Direction: %s\n", state.Behaviour, state.Floor, state.Direction)
		fmt.Println("  Cab Requests:")
		for i, req := range state.CabRequests {
			fmt.Printf("    Floor%d - %s\n", i, reqToString(req))
		}
	}

	fmt.Println("Hall Requests:")
	// Reverse iteration of the HallRequests with a single loop
	for i := len(elevatorSystem.HallRequests) - 1; i >= 0; i-- {
		fmt.Printf("    Floor%d: ", i)
		for _, req := range elevatorSystem.HallRequests[i] {
			fmt.Printf("%s ", reqToString(req))
		}
		fmt.Println()
	}
}
*/
// String method to print the RequestStatus as a string
func ReqToString(req int) string {
	switch req {
	case unknownOrder:
		return "unknown"
	case noOrder:
		return "no request"
	case unconfirmedOrder:
		return "unconfirmed"
	case confirmedOrder:
		return "confirmed"
	case servicedOrder:
		return "serviced order"
	default:
		return "Invalid"
	}
}
