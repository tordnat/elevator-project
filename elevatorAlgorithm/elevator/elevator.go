package elevator

import (
	"elevatorDriver/elevio"
)

const N_FLOORS = 4
const N_BUTTONS = 3
const N_HALL_BUTTONS = 2
const DOOR_OPEN_DURATION_S = 2

type ElevatorBehaviour int

const (
	EB_Idle ElevatorBehaviour = iota
	EB_DoorOpen
	EB_Moving
)

type ElevatorState struct {
	Behaviour ElevatorBehaviour
	Floor     int
	Direction elevio.MotorDirection
	Requests  [][]bool //This should be filled from HRA
}

type Order elevio.ButtonEvent

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
