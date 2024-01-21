package elevatorInterface

import (
	"elevio"
)

type ElevatorInput interface {
	FloorSensor() int
	RequestButton(floor int, b elevio.ButtonType) int
	StopButton() bool
	Obstruction() bool
}

type ElevatorOutput interface {
	FloorIndicator()
	GetButton(button elevio.ButtonType, floor int) bool
}
