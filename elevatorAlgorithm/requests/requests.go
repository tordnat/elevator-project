package requests

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/hra"
	"elevatorDriver/elevio"
	"log"
)

type DirnBehaviourPair struct {
	Dir       elevio.MotorDirection
	Behaviour elevator.ElevatorBehaviour
}

// This is ambiguous, change the name
func requestsAbove(e hra.LocalElevatorState, confirmedOrders [][]bool) bool {
	for f := e.Floor + 1; f < elevator.N_FLOORS; f++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			if confirmedOrders[f][btn] {
				return true
			}
		}
	}
	return false
}

// This is ambiguous, change the name
func requestsBelow(e hra.LocalElevatorState, confirmedOrders [][]bool) bool {
	for f := 0; f < e.Floor; f++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			if confirmedOrders[f][btn] {
				return true
			}
		}
	}
	return false
}

func requestsHere(e hra.LocalElevatorState, confirmedOrders [][]bool) bool {
	for btn := 0; btn < elevator.N_BUTTONS; btn++ {
		if confirmedOrders[e.Floor][elevio.BT_Cab] {
			return true
		}
	}
	return false
}

func ChooseDirection(e hra.LocalElevatorState, confirmedOrders [][]bool) DirnBehaviourPair {
	switch e.Direction {
	case elevio.MD_Up:
		if requestsAbove(e, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
		} else if requestsHere(e, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_DoorOpen}
		} else if requestsBelow(e, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
		} else {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
		}
	case elevio.MD_Down:
		if requestsBelow(e, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
		} else if requestsHere(e, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_DoorOpen}
		} else if requestsAbove(e, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
		} else {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
		}
	case elevio.MD_Stop:
		if requestsHere(e, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_DoorOpen}
		} else if requestsAbove(e, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
		} else if requestsBelow(e, confirmedOrders) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
		} else {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
		}
	default:
		return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}

	}
}

func ShouldStop(e hra.LocalElevatorState, confirmedOrders [][]bool) bool {
	switch e.Direction {
	case elevio.MD_Down:
		return confirmedOrders[e.Floor][elevio.BT_HallDown] ||
			confirmedOrders[e.Floor][elevio.BT_Cab] ||
			!requestsBelow(e, confirmedOrders)
	case elevio.MD_Up:
		return confirmedOrders[e.Floor][elevio.BT_HallUp] ||
			confirmedOrders[e.Floor][elevio.BT_Cab] ||
			!requestsAbove(e, confirmedOrders)
	default:
		return true
	}
}

func ShouldClearImmediately(e hra.LocalElevatorState, floor int, btn_type elevio.ButtonType) bool {
	return e.Floor == floor &&
		(((e.Direction == elevio.MD_Up) && (btn_type == elevio.BT_HallUp)) ||
			((e.Direction == elevio.MD_Down) && (btn_type == elevio.BT_HallDown)) ||
			(e.Direction == elevio.MD_Stop) ||
			(btn_type == elevio.BT_Cab))
}

func ClearAtCurrentFloor(e hra.LocalElevatorState, confirmedOrders [][]bool) hra.LocalElevatorState {
	log.Println("In ClearAtCurrentFloor")
	confirmedOrders[e.Floor][elevio.BT_Cab] = false
	switch e.Direction {
	case elevio.MD_Up:
		if !requestsAbove(e, confirmedOrders) && !confirmedOrders[e.Floor][elevio.BT_Cab] {
			confirmedOrders[e.Floor][elevio.BT_Cab] = false
		}
		confirmedOrders[e.Floor][elevio.BT_Cab] = false
	case elevio.MD_Down:
		if !requestsBelow(e, confirmedOrders) && !confirmedOrders[e.Floor][elevio.BT_HallDown] {
			confirmedOrders[e.Floor][elevio.BT_Cab] = false
		}
		confirmedOrders[e.Floor][elevio.BT_Cab] = false
	default:
		confirmedOrders[e.Floor][elevio.BT_Cab] = false
		confirmedOrders[e.Floor][elevio.BT_HallDown] = false
	}
	return e
}
