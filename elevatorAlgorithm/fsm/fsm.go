package fsm

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/hra"
	"elevatorAlgorithm/requests"
	"elevatorAlgorithm/timer"
	"elevatorDriver/elevio"
	"log"
)

/// TODO: Use type OrderAssignments map[string][][]bool
/// instead of elevatorSystem.hra ?? +

func setAllLights(confirmedOrders [][]bool) {
	for floor := 0; floor < elevator.N_FLOORS; floor++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, confirmedOrders[floor][btn])
		}
	}
}

func OnInitBetweenFloors(e hra.LocalElevatorState) hra.LocalElevatorState {
	elevio.SetMotorDirection(elevio.MD_Down)
	e.Direction = elevio.MD_Down
	e.Behaviour = elevator.EB_Moving
	return e
}

func UpdateButtonRequest(elevatorSystem hra.ElevatorSystem, buttonEvent elevio.ButtonEvent) hra.ElevatorSystem {
	elevatorSystem.HallRequests[buttonEvent.Floor][buttonEvent.Button] = 1
	return elevatorSystem
}

func UpdateFloor(id string, elevatorSystem hra.ElevatorSystem, floor int) hra.ElevatorSystem {
	modifiedElevatorState := elevatorSystem.ElevatorStates[id]
	modifiedElevatorState.Floor = floor
	elevatorSystem.ElevatorStates[id] = modifiedElevatorState
	return elevatorSystem
}

// Consider encapsulating the variable currentElevatorSystem
// WARNING: This module assumes only one floor/button changes at a time
func Transition(id string, currentElevatorSystem hra.ElevatorSystem, updatedElevatorSystem hra.ElevatorSystem, confirmedOrders [][]bool) hra.ElevatorSystem {
	if currentElevatorSystem.ElevatorStates[id].Floor != updatedElevatorSystem.ElevatorStates[id].Floor {
		log.Println("Floors changed")
		currentElevatorSystem = OnFloorArrival(id, updatedElevatorSystem, confirmedOrders)
	} else {
		var updatedButton elevio.ButtonEvent
		buttonIsChanged := false
		for floor := 0; floor < elevator.N_FLOORS; floor++ {
			for btn := 0; btn < elevator.N_HALL_BUTTONS; btn++ {
				if currentElevatorSystem.HallRequests[floor][btn] != updatedElevatorSystem.HallRequests[floor][btn] {
					buttonIsChanged = true
					updatedButton.Button = elevio.ButtonType(btn)
					updatedButton.Floor = floor
				}
			}
			if currentElevatorSystem.ElevatorStates[id].CabRequests[floor] != updatedElevatorSystem.ElevatorStates[id].CabRequests[floor] {
				buttonIsChanged = true
			}

		}
		if buttonIsChanged {
			currentElevatorSystem = OnRequestButtonPress(id, updatedElevatorSystem, confirmedOrders, updatedButton)
		}
	}
	return currentElevatorSystem
}

func OnRequestButtonPress(id string, elevatorSystem hra.ElevatorSystem, confimedOrders [][]bool, buttonEvent elevio.ButtonEvent) hra.ElevatorSystem {
	modifiedElevatorState := elevatorSystem.ElevatorStates[id]

	switch modifiedElevatorState.Behaviour {
	case elevator.EB_DoorOpen:
		if requests.ShouldClearImmediately(modifiedElevatorState, buttonEvent.Floor, buttonEvent.Button) {
			confimedOrders[buttonEvent.Floor][buttonEvent.Button] = false
			log.Println("Clearing order!")
			timer.Start()
		}
	case elevator.EB_Idle:
		pair := requests.ChooseDirection(modifiedElevatorState, confimedOrders)

		modifiedElevatorState.Direction = pair.Dir // Change to Direction instead of Dir
		modifiedElevatorState.Behaviour = pair.Behaviour

		switch pair.Behaviour {
		case elevator.EB_DoorOpen:
			elevio.SetDoorOpenLamp(true)
			timer.Start()
			confimedOrders[buttonEvent.Floor][buttonEvent.Button] = false
			log.Println("Clearing order!")
		case elevator.EB_Moving:
			elevio.SetMotorDirection(modifiedElevatorState.Direction)
		}
	}
	setAllLights(confimedOrders)
	elevatorSystem.ElevatorStates[id] = modifiedElevatorState
	return elevatorSystem

}

func OnFloorArrival(id string, elevatorSystem hra.ElevatorSystem, confimedOrders [][]bool) hra.ElevatorSystem {
	log.Println("On floor arrival")
	modifiedElevatorState := elevatorSystem.ElevatorStates[id]
	switch modifiedElevatorState.Behaviour {
	case elevator.EB_Moving:
		if requests.ShouldStop(modifiedElevatorState, confimedOrders) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			modifiedElevatorState = requests.ClearAtCurrentFloor(modifiedElevatorState, confimedOrders)
			setAllLights(confimedOrders)
			elevio.SetDoorOpenLamp(true)
			timer.Start()
			modifiedElevatorState.Behaviour = elevator.EB_DoorOpen
		}
	}
	elevatorSystem.ElevatorStates[id] = modifiedElevatorState
	return elevatorSystem
}

func OnDoorTimeOut(id string, elevatorSystem hra.ElevatorSystem, confirmedOrders [][]bool) hra.LocalElevatorState {
	modifiedElevatorState := elevatorSystem.ElevatorStates[id]
	switch modifiedElevatorState.Behaviour {
	case elevator.EB_DoorOpen:
		pair := requests.ChooseDirection(modifiedElevatorState, confirmedOrders)
		modifiedElevatorState.Direction = pair.Dir
		modifiedElevatorState.Behaviour = pair.Behaviour

		switch modifiedElevatorState.Behaviour {
		case elevator.EB_DoorOpen:
			timer.Start()
			modifiedElevatorState = requests.ClearAtCurrentFloor(modifiedElevatorState, confirmedOrders) // This should be fine since orders are confirmed
			setAllLights(confirmedOrders)
		case elevator.EB_Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(modifiedElevatorState.Direction)
		case elevator.EB_Moving:
			elevio.SetDoorOpenLamp(false)
		}
	}
	return modifiedElevatorState
}
