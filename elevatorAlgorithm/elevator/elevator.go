package elevator

import (
	"elevatorDriver/elevio"
	"time"
)

const N_FLOORS = 4
const N_BUTTONS = 3
const N_HALL_BUTTONS = 2
const DOOR_OPEN_DURATION = 3 * time.Second

type ElevatorBehaviour int

const (
	EB_Idle ElevatorBehaviour = iota
	EB_DoorOpen
	EB_Moving
)

type Elevator struct {
	Behaviour ElevatorBehaviour
	Floor     int
	Direction elevio.MotorDirection
	Orders    [][]bool //This should be filled from HRA
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

func NewElevator(behaviour ElevatorBehaviour, floor int, dir elevio.MotorDirection, numFloors int) Elevator {
	newElevator := Elevator{}
	newElevator.Behaviour = behaviour
	newElevator.Floor = floor
	newElevator.Direction = dir
	newElevator.Orders = make([][]bool, numFloors)
	for i := 0; i < numFloors; i++ {
		newElevator.Orders[i] = make([]bool, N_BUTTONS)
	}
	return newElevator
}
