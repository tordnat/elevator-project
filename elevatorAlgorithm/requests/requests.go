package requests

import (
	"elevatorAlgorithm/elevator"
	"elevatorDriver/elevio"
)

type DirnBehaviourPair struct {
	Dir       elevio.MotorDirection
	Behaviour elevator.ElevatorBehaviour
}

func requestsAbove(currentFloor int, confirmedOrders [][]bool) bool {
	for f := currentFloor + 1; f < elevator.N_FLOORS; f++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			if confirmedOrders[f][btn] {
				return true
			}
		}
	}
	return false
}

func requestsBelow(currentFloor int, confirmedOrders [][]bool) bool {
	for f := 0; f < currentFloor; f++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			if confirmedOrders[f][btn] {
				return true
			}
		}
	}
	return false
}

func requestsHere(currentFloor int, confirmedOrders [][]bool) bool {
	for btn := 0; btn < elevator.N_BUTTONS; btn++ {
		if confirmedOrders[currentFloor][elevio.BT_Cab] {
			return true
		}
	}
	return false
}

func ChooseDirection(direction elevio.MotorDirection, floor int, confirmedOrders [][]bool) DirnBehaviourPair {
	switch direction {
	case elevio.MD_Up:
		if requestsAbove(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
		} else if requestsHere(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_DoorOpen}
		} else if requestsBelow(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
		} else {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
		}
	case elevio.MD_Down:
		if requestsBelow(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
		} else if requestsHere(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_DoorOpen}
		} else if requestsAbove(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
		} else {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
		}
	case elevio.MD_Stop:
		if requestsHere(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_DoorOpen}
		} else if requestsAbove(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
		} else if requestsBelow(floor, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
		} else {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
		}
	default:
		return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}

	}
}

func ShouldStop(direction elevio.MotorDirection, floor int, confirmedOrders [][]bool) bool {
	switch direction {
	case elevio.MD_Down:
		return confirmedOrders[floor][elevio.BT_HallDown] ||
			confirmedOrders[floor][elevio.BT_Cab] ||
			!requestsBelow(floor, confirmedOrders)
	case elevio.MD_Up:
		return confirmedOrders[floor][elevio.BT_HallUp] ||
			confirmedOrders[floor][elevio.BT_Cab] ||
			!requestsAbove(floor, confirmedOrders)
	default:
		return true
	}
}

func ShouldClearImmediately(currentFloor int, currentDir elevio.MotorDirection, orderEvent elevator.Order) bool {
	return currentFloor == orderEvent.Floor &&
		(((currentDir == elevio.MD_Up) && (orderEvent.Button == elevio.BT_HallUp)) ||
			((currentDir == elevio.MD_Down) && (orderEvent.Button == elevio.BT_HallDown)) ||
			(currentDir == elevio.MD_Stop) ||
			(orderEvent.Button == elevio.BT_Cab))
}

func ClearCab(floor int) elevator.Order {
	var orderToClear elevator.Order
	orderToClear.Button = elevio.BT_Cab
	orderToClear.Floor = floor
	return orderToClear
}

func ClearHallUp(e elevator.ElevatorState) elevator.Order {
	var orderToClear elevator.Order
	orderToClear.Floor = e.Floor
	orderToClear.Button = elevio.BT_HallUp

	switch e.Direction {
	case elevio.MD_Up:
		return orderToClear

	case elevio.MD_Down:
		if !requestsBelow(e.Floor, e.Requests) && !e.Requests[e.Floor][elevio.BT_HallDown] {
			return orderToClear
		}

	default:
		return orderToClear
	}

	return elevator.Order{-1, -1}
}

func ClearHallDown(e elevator.ElevatorState) elevator.Order {
	var orderToClear elevator.Order
	orderToClear.Floor = e.Floor
	orderToClear.Button = elevio.BT_HallDown

	switch e.Direction {
	case elevio.MD_Up:
		if !requestsAbove(e.Floor, e.Requests) && !e.Requests[e.Floor][elevio.BT_HallUp] {
			return orderToClear
		}
	case elevio.MD_Down:
		return orderToClear
	default:
		return orderToClear
	}

	return elevator.Order{-1, -1}
}
