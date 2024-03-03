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
func requestsAbove(e hra.LocalElevatorState, hr hra.HallRequestsType) bool {
	for f := e.Floor + 1; f < elevator.N_FLOORS; f++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			if hr[f][btn] {
				return true
			}
		}
	}
	return false
}

// This is ambiguous, change the name
func requestsBelow(e hra.LocalElevatorState, hr hra.HallRequestsType) bool {
	for f := 0; f < e.Floor; f++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			if hr[f][btn] {
				return true
			}
		}
	}
	return false
}

func requestsHere(e hra.LocalElevatorState, hr hra.HallRequestsType) bool {
	for btn := 0; btn < elevator.N_BUTTONS; btn++ {
		if hr[e.Floor][elevio.BT_Cab] {
			return true
		}
	}
	return false
}

func ChooseDirection(e hra.LocalElevatorState, hr hra.HallRequestsType) DirnBehaviourPair {
	switch e.Direction {
	case elevio.MD_Up:
		if requestsAbove(e, hr) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
		} else if requestsHere(e, hr) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_DoorOpen}
		} else if requestsBelow(e, hr) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
		} else {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
		}
	case elevio.MD_Down:
		if requestsBelow(e, hr) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
		} else if requestsHere(e, hr) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_DoorOpen}
		} else if requestsAbove(e, hr) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
		} else {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
		}
	case elevio.MD_Stop:
		if requestsHere(e, hr) {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_DoorOpen}
		} else if requestsAbove(e, hr) {
			return DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
		} else if requestsBelow(e, hr) {
			return DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
		} else {
			return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
		}
	default:
		return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}

	}
}

func ShouldStop(e hra.LocalElevatorState, hr hra.HallRequestsType) bool {
	switch e.Direction {
	case elevio.MD_Down:
		return hr[e.Floor][elevio.BT_HallDown] ||
			hr[e.Floor][elevio.BT_Cab] ||
			!requestsBelow(e, hr)
	case elevio.MD_Up:
		return hr[e.Floor][elevio.BT_HallUp] ||
			hr[e.Floor][elevio.BT_Cab] ||
			!requestsAbove(e, hr)
	default:
		return true
	}
}

func ShouldClearImmediately(e hra.LocalElevatorState, btn_floor int, btn_type elevio.ButtonType) bool {
	return e.Floor == btn_floor &&
		(((e.Direction == elevio.MD_Up) && (btn_type == elevio.BT_HallUp)) ||
			((e.Direction == elevio.MD_Down) && (btn_type == elevio.BT_HallDown)) ||
			(e.Direction == elevio.MD_Stop) ||
			(btn_type == elevio.BT_Cab))
}

func ClearAtCurrentFloor(e hra.LocalElevatorState, hr hra.HallRequestsType) hra.LocalElevatorState {
	log.Println("In ClearAtCurrentFloor")
	hr[e.Floor][elevio.BT_Cab] = false
	switch e.Direction {
	case elevio.MD_Up:
		if !requestsAbove(e, hr) && !hr[e.Floor][elevio.BT_Cab] {
			hr[e.Floor][elevio.BT_Cab] = false
		}
		hr[e.Floor][elevio.BT_Cab] = false
	case elevio.MD_Down:
		if !requestsBelow(e, hr) && !hr[e.Floor][elevio.BT_HallDown] {
			hr[e.Floor][elevio.BT_Cab] = false
		}
		hr[e.Floor][elevio.BT_Cab] = false
	default:
		hr[e.Floor][elevio.BT_Cab] = false
		hr[e.Floor][elevio.BT_HallDown] = false
	}
	return e
}
