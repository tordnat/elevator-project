package fsm

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/hra"
	"elevatorAlgorithm/requests"
	"elevatorDriver/elevio"
)


func setAllLights(system hra.ElevatorSystem , hr hra.HallRequests) {
	for floor := 0; floor < elevator.N_FLOORS; floor++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
	

func OnInitBetweenFloors() {
	elevio.SetMotorDirection(elevio.MD_Down)
	elevatorSingelton.Dirn = elevio.MD_Down
	elevatorSingelton.Behaviour = elevator.EB_Moving
}

func OnRequestButtonPress(btnFloor int, btnType elevio.ButtonType) {
	//fmt.Printf("\n\n%s(%d, %s)\n", function, btnFloor, elevioButtonToString(btnType))
	//log.Println("Pressed button for floor ", btnFloor)
	switch elevatorSingelton.Behaviour {
	case elevator.EB_DoorOpen:
		if requests.ShouldClearImmediately(elevatorSingelton, btnFloor, btnType) {
			timer.Start()
		} else {
			elevatorSingelton.Requests[btnFloor][btnType] = true
		}
		break
	case elevator.EB_Moving:
		elevatorSingelton.Requests[btnFloor][btnType] = true
		break
	case elevator.EB_Idle:
		elevatorSingelton.Requests[btnFloor][btnType] = true
		pair := requests.ChooseDirection(elevatorSingelton)

		elevatorSingelton.Dirn = pair.Dir
		elevatorSingelton.Behaviour = pair.Behaviour

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
