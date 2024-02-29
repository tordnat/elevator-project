package fsm

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/hra"
	"elevatorAlgorithm/requests"
	"elevatorAlgorithm/timer"
	"elevatorDriver/elevio"
)

func setAllLights(confirmedOrders [][]bool) {
	for floor := 0; floor < elevator.N_FLOORS; floor++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, confirmedOrders[floor][btn])
		}
	}
}

func OnInitBetweenFloors(e hra.LocalElevatorState) {
	elevio.SetMotorDirection(elevio.MD_Down)
	e.Direction = elevio.MD_Down
	e.Behaviour = elevator.EB_Moving
}

func OnRequestButtonPress(btnFloor int, btnType elevio.ButtonType, e hra.LocalElevatorState, confirmedOrders [][]bool) {
	//fmt.Printf("\n\n%s(%d, %s)\n", function, btnFloor, elevioButtonToString(btnType))
	//log.Println("Pressed button for floor ", btnFloor)
	switch e.Behaviour {
	case elevator.EB_DoorOpen:
		if requests.ShouldClearImmediately(e, btnFloor, btnType) {
			timer.Start()
		} else {
			//hr[btnFloor][btnType] = true
		}
		break
	case elevator.EB_Moving:
		confirmedOrders[btnFloor][btnType] = true
		break
	case elevator.EB_Idle:
		//hr[btnFloor][btnType] = true
		pair := requests.ChooseDirection(e)

		e.Direction = pair.Dir
		e.Behaviour = pair.Behaviour

		switch pair.Behaviour {
		case elevator.EB_DoorOpen:
			elevio.SetDoorOpenLamp(true)
			timer.Start()
			elevatorSingelton = requests.ClearAtCurrentFloor(elevatorSingelton)
		case elevator.EB_Moving:
			elevio.SetMotorDirection(elevatorSingelton.Dirn)
		}
	}
	setAllLights(elevatorSingelton)
}

func OnFloorArrival(newFloor int) {
	// elevator.elevator_print(elevatorSingelton)
	elevatorSingelton.Floor = newFloor
	elevio.SetFloorIndicator(elevatorSingelton.Floor)

	switch elevatorSingelton.Behaviour {
	case elevator.EB_Moving:
		if requests.ShouldStop(elevatorSingelton) {
			//log.Println("Arrived at floor: ", elevatorSingelton.Floor)
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevio.SetDoorOpenLamp(true)
			elevatorSingelton = requests.ClearAtCurrentFloor(elevatorSingelton)
			timer.Start()
			setAllLights(elevatorSingelton)
			elevatorSingelton.Behaviour = elevator.EB_DoorOpen
		}
	}
	//log. // Consider proper logging
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
			timer.Start()
			elevatorSingelton = requests.ClearAtCurrentFloor(elevatorSingelton)
			setAllLights(elevatorSingelton)
		case elevator.EB_Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevatorSingelton.Dirn)
		}
	}
}
