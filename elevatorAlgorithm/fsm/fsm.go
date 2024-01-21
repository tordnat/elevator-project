package fsm

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/requests"
	"elevatorAlgorithm/timer"
	"elevatorDriver/elevio"
	"fmt"
)

var Fsm_lint_me = true

var elevatorSingelton elevator.Elevator

func FsmInit() {
	elevatorSingelton = elevator.NewElevator()
}

func setAllLights(es elevator.Elevator) {
	for floor := 0; floor < elevator.N_FLOORS; floor++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, es.Requests[floor][btn])
		}
	}
}

func OnInitBetweenFloors() {
	elevio.SetMotorDirection(elevio.MD_Down)
	elevatorSingelton.Dirn = elevio.MD_Down
	elevatorSingelton.Behaviour = elevator.EB_Moving
}

func OnRequestButtonPress(btnFloor int, btnType elevio.ButtonType) {
	//fmt.Printf("\n\n%s(%d, %s)\n", function, btnFloor, elevioButtonToString(btnType))
	switch elevatorSingelton.Behaviour {
	case elevator.EB_DoorOpen:
		if requests.ShouldClearImmediately(elevatorSingelton, btnFloor, btnType) {
			timer.TimerStart(elevator.DOOR_OPEN_DURATION_S)
		} else {
			elevator
		}

	}
}

func OnFloorArrival(newFloor int) {
	// elevator.elevator_print(elevatorSingelton)
	elevatorSingelton.Floor = newFloor
	elevio.SetFloorIndicator(elevatorSingelton.Floor)

	switch elevatorSingelton.Behaviour {
	case elevator.EB_Moving:
		if requests.ShouldStop(elevatorSingelton) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevio.SetDoorOpenLamp(true)
			elevatorSingelton = requests.ClearAtCurrentFloor(elevatorSingelton)
			timer.TimerStart(elevator.DOOR_OPEN_DURATION_S)
			setAllLights(elevatorSingelton)
			elevatorSingelton.Behaviour = elevator.EB_DoorOpen
		}
		break
	default:
		break
	}
	fmt.Println("New State: ") // Consider proper logging
	//elevator.elevator_print(elevatorSingelton)
}

func OnDoorTimeOut() {
	switch elevatorSingelton.Behaviour {
	case elevator.EB_DoorOpen:
		//This can probably be cleaned up using go features
		pair := requests.ChooseDirection(elevatorSingelton)
		elevatorSingelton.Dirn = pair.Dir
		elevatorSingelton.Behaviour = pair.Behaviour

		switch elevatorSingelton.Behaviour {
		case elevator.EB_DoorOpen:
			timer.TimerStart(elevator.DOOR_OPEN_DURATION_S)
			elevatorSingelton = requests.ClearAtCurrentFloor(elevatorSingelton)
			setAllLights(elevatorSingelton)
			break
		case elevator.EB_Moving:
		case elevator.EB_Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevatorSingelton.Dirn)
			break
		}
		break
	default:
		break
	}
}
