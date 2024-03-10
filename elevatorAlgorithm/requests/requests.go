package requests

import (
	"elevatorAlgorithm/elevator"
	"elevatorDriver/elevio"
)

type DirnBehaviourPair struct {
	Dir       elevio.MotorDirection
	Behaviour elevator.ElevatorBehaviour
}
type ClearFloorOrders struct {
	Floor    int
	HallUp   bool
	HallDown bool
	Cab      bool
}

func ChooseDirection(direction elevio.MotorDirection, floor int, confirmedOrders [][]bool) DirnBehaviourPair {
	switch direction {
	case elevio.MD_Up:
		if RequestsAbove(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
		} else if RequestsHere(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_DoorOpen}
		} else if RequestsBelow(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
		} else {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
		}
	case elevio.MD_Down:
		if RequestsBelow(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
		} else if RequestsHere(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_DoorOpen}
		} else if RequestsAbove(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
		} else {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
		}
	case elevio.MD_Stop:
		if RequestsHere(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_DoorOpen}
		} else if RequestsAbove(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
		} else if RequestsBelow(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
		} else {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
		}
	default:
		return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}

	}
}

func RequestsAbove(currentFloor int, confirmedOrders [][]bool) bool {
	for f := currentFloor + 1; f < elevator.N_FLOORS; f++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			if confirmedOrders[f][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsBelow(currentFloor int, confirmedOrders [][]bool) bool {
	for f := 0; f < currentFloor; f++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			if confirmedOrders[f][btn] {
				return true
			}
		}
	}
	return false
}

func RequestsHere(currentFloor int, confirmedOrders [][]bool) bool {
	for btn := 0; btn < elevator.N_BUTTONS; btn++ {
		if confirmedOrders[currentFloor][elevio.BT_Cab] {
			return true
		}
	}
	return false
}

func HaveOrders(currentFloor int, confirmedOrders [][]bool) bool {
	if RequestsHere(currentFloor, confirmedOrders) {
		return true
	}
	if RequestsAbove(currentFloor, confirmedOrders) {
		return true
	}
	if RequestsBelow(currentFloor, confirmedOrders) {
		return true
	}
	return false
}

func ShouldStop(direction elevio.MotorDirection, floor int, confirmedOrders [][]bool) bool {
	switch direction {
	case elevio.MD_Down:
		return confirmedOrders[floor][elevio.BT_HallDown] ||
			confirmedOrders[floor][elevio.BT_Cab] ||
			!RequestsBelow(floor, confirmedOrders)
	case elevio.MD_Up:
		return confirmedOrders[floor][elevio.BT_HallUp] ||
			confirmedOrders[floor][elevio.BT_Cab] ||
			!RequestsAbove(floor, confirmedOrders)
	default:
		return true
	}
}

func ClearAtFloor(currentFloor int, currentDir elevio.MotorDirection, orders [][]bool) ClearFloorOrders {
	orderToClear := ClearFloorOrders{currentFloor, false, false, false}
	for btn := range orders[currentFloor] {
		if (currentDir == elevio.MD_Up) && (elevio.ButtonType(btn) == elevio.BT_HallUp) {
			orderToClear.HallUp = true
		}
		if (currentDir == elevio.MD_Down) && (elevio.ButtonType(btn) == elevio.BT_HallDown) {
			orderToClear.HallDown = true
		}
		if currentDir == elevio.MD_Stop {
			orderToClear.Cab = true
			orderToClear.HallDown = true
			orderToClear.HallUp = true
		}
		if elevio.ButtonType(btn) == elevio.BT_Cab {
			orderToClear.Cab = true
		}
	}
	return orderToClear
}

func ShouldClearHallUp(floor int, dir elevio.MotorDirection, requests [][]bool) bool {
	if dir == elevio.MD_Down {
		if RequestsBelow(floor, requests) || requests[floor][elevio.BT_HallDown] {
			return false
		}
	}
	return true
}

func ShouldClearHallDown(floor int, dir elevio.MotorDirection, requests [][]bool) bool {
	if dir == elevio.MD_Up {
		if RequestsAbove(floor, requests) || requests[floor][elevio.BT_HallUp] {
			return false
		}
	}
	return true
}
