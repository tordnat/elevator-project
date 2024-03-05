package fsm

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/requests"
	"elevatorAlgorithm/timer"
	"elevatorDriver/elevio"
	"log"
)

// IMPORTANT: functions should either be completly pure,
// or they should communicate using channels and hardware.
// E.g execution of actions vs mutating data
// State transition functions may be the only exception, they return the new state. We could change the functions to be pure and return a list of HW actions, but a bit bloated?
// There is no need to use terminology like confirmed orders here. Alle orders are bools in the fsm, therefore they are simply orders.

func FSM(orderAssignment chan elevator.Order, orderCompleted chan elevator.Order, floorEvent chan int, obstructionEvent chan bool) {
	elevState := elevator.ElevatorState{elevator.EB_Idle, -1, elevio.MD_Stop, make([][]bool, 3*4)} //This 2d array should be properly filled
	//Get elevState to a defined state here
	select {
	case event := <-orderAssignment: //Analogue of button press
		log.Println("New assignemtn ", event)
		elevState = OnRequestButtonPress(elevState, event)

	case event := <-floorEvent:
		log.Println("Sending", event)
		elevState = OnFloorArrival(elevState)

	case <-timer.TimerChan:
		log.Println("Door timeout")
		timer.Timedout = true
		OnDoorTimeOut(elevState)

	case <-obstructionEvent:
		if elevState.Behaviour == elevator.EB_DoorOpen {
			timer.Start()
		}
	}
}

func setAllLights(confirmedOrders [][]bool) {
	for floor := 0; floor < elevator.N_FLOORS; floor++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, confirmedOrders[floor][btn])
		}
	}
}

func OnInitBetweenFloors(e elevator.ElevatorState) elevator.ElevatorState {
	elevio.SetMotorDirection(elevio.MD_Down)
	e.Direction = elevio.MD_Down
	e.Behaviour = elevator.EB_Moving
	return e
}

// Can this handle multiple order reassignments. Looks like it, so should work with HRA
func OnRequestButtonPress(elevState elevator.ElevatorState, orderEvent elevator.Order) elevator.ElevatorState {
	switch elevState.Behaviour {
	case elevator.EB_DoorOpen:
		if requests.ShouldClearImmediately(elevState.Floor, elevState.Direction, orderEvent) {
			//Clear order by channel here (spesific order)
			log.Println("Clearing order!")
			timer.Start()
		}
	case elevator.EB_Idle:
		pair := requests.ChooseDirection(elevState.Direction, elevState.Floor, elevState.Requests)

		elevState.Direction = pair.Dir
		elevState.Behaviour = pair.Behaviour

		switch pair.Behaviour {
		case elevator.EB_DoorOpen:
			elevio.SetDoorOpenLamp(true)
			timer.Start()
			//Clear order by channel here (ClearAtCurrentFloor)
			log.Println("Clearing order!")
		case elevator.EB_Moving:
			elevio.SetMotorDirection(elevState.Direction)
		}
	}
	setAllLights(elevState.Requests)
	return elevState

}

func OnFloorArrival(elevState elevator.ElevatorState) elevator.ElevatorState {
	log.Println("On floor arrival")
	switch elevState.Behaviour {
	case elevator.EB_Moving:
		if requests.ShouldStop(elevState.Direction, elevState.Floor, elevState.Requests) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			//Clear order by channel here (ClearAtCurrentFloor)
			setAllLights(elevState.Requests)
			elevio.SetDoorOpenLamp(true)
			timer.Start()
			elevState.Behaviour = elevator.EB_DoorOpen
		}
	}
	return elevState
}

func OnDoorTimeOut(elevState elevator.ElevatorState) {
	switch elevState.Behaviour {
	case elevator.EB_DoorOpen:
		pair := requests.ChooseDirection(elevState.Direction, elevState.Floor, elevState.Requests)
		elevState.Direction = pair.Dir
		elevState.Behaviour = pair.Behaviour

		switch elevState.Behaviour {
		case elevator.EB_DoorOpen:
			timer.Start()
			//Clear order by channel here (ClearAtCurrentFloor)
			setAllLights(elevState.Requests)
		case elevator.EB_Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevState.Direction)
		case elevator.EB_Moving:
			elevio.SetDoorOpenLamp(false)
		}
	}
}
