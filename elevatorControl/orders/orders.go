package orders

import (
	"elevatorControl/elevator"
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
		if OrdersAbove(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
		} else if OrdersHere(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_DoorOpen}
		} else if OrdersBelow(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
		} else {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
		}
	case elevio.MD_Down:
		if OrdersBelow(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
		} else if OrdersHere(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_DoorOpen}
		} else if OrdersAbove(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
		} else {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
		}
	case elevio.MD_Stop:
		if OrdersHere(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_DoorOpen}
		} else if OrdersAbove(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
		} else if OrdersBelow(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
		} else {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
		}
	default:
		return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}

	}
}

func OrdersAbove(currentFloor int, confirmedOrders [][]bool) bool {
	for f := currentFloor + 1; f < elevator.N_FLOORS; f++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			if confirmedOrders[f][btn] {
				return true
			}
		}
	}
	return false
}

func OrdersBelow(currentFloor int, confirmedOrders [][]bool) bool {
	for f := 0; f < currentFloor; f++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			if confirmedOrders[f][btn] {
				return true
			}
		}
	}
	return false
}

func OrdersHere(currentFloor int, confirmedOrders [][]bool) bool {
	for btn := 0; btn < elevator.N_BUTTONS; btn++ {
		if confirmedOrders[currentFloor][btn] {
			return true
		}
	}
	return false
}

func HaveOrders(currentFloor int, confirmedOrders [][]bool) bool {
	if OrdersHere(currentFloor, confirmedOrders) {
		return true
	}
	if OrdersAbove(currentFloor, confirmedOrders) {
		return true
	}
	if OrdersBelow(currentFloor, confirmedOrders) {
		return true
	}
	return false
}

func ShouldStop(direction elevio.MotorDirection, floor int, confirmedOrders [][]bool) bool {
	switch direction {
	case elevio.MD_Down:
		return confirmedOrders[floor][elevio.BT_HallDown] ||
			confirmedOrders[floor][elevio.BT_Cab] ||
			!OrdersBelow(floor, confirmedOrders)
	case elevio.MD_Up:
		return confirmedOrders[floor][elevio.BT_HallUp] ||
			confirmedOrders[floor][elevio.BT_Cab] ||
			!OrdersAbove(floor, confirmedOrders)
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

func ShouldClearHallUp(floor int, dir elevio.MotorDirection, orders [][]bool) bool {
	if dir == elevio.MD_Down {
		if OrdersBelow(floor, orders) || orders[floor][elevio.BT_HallDown] {
			return false
		}
	}
	return true
}

func ShouldClearHallDown(floor int, dir elevio.MotorDirection, orders [][]bool) bool {
	if dir == elevio.MD_Up {
		if OrdersAbove(floor, orders) || orders[floor][elevio.BT_HallUp] {
			return false
		}
	}
	return true
}
