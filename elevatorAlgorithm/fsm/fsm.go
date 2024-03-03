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

func OnInitBetweenFloors(e hra.LocalElevatorState) {
	elevio.SetMotorDirection(elevio.MD_Down)
	e.Direction = elevio.MD_Down
	e.Behaviour = elevator.EB_Moving
}

func UpdateButtonRequest(elevatorSystem hra.ElevatorSystem, buttonEvent elevio.ButtonEvent) hra.ElevatorSystem {
	elevatorSystem.HallRequests[buttonEvent.Floor][buttonEvent.Button] = true
	return elevatorSystem
}

func UpdateFloor(id string, elevatorSystem hra.ElevatorSystem, floor int) hra.ElevatorSystem {
	modifiedElevatorState := elevatorSystem.ElevatorStates[id]
	modifiedElevatorState.Floor = floor
	elevatorSystem.ElevatorStates[id] = modifiedElevatorState
	return elevatorSystem
}

func Transition(id string, currentElevatorSystem hra.ElevatorSystem, updatedElevatorSystem hra.ElevatorSystem) hra.ElevatorSystem {
	// TODO:
	// - check for change in floor, button or door
	// - execute transition for each case using a separate module
	// - do necessary HW stuff: Lights, motor etc.
	if currentElevatorSystem.ElevatorStates[id].Floor != updatedElevatorSystem.ElevatorStates[id].Floor {
		currentElevatorSystem = OnFloorArrival(id, updatedElevatorSystem)
	} else if (currentElevatorSystem.ElevatorStates[id].Behaviour == elevator.EB_DoorOpen & timer.Timedout ) {
		
	}
	
	else {
		var updatedButton elevio.ButtonEvent
		buttonIsChanged := false
		for floor := 0; floor < elevator.N_FLOORS; floor++ {
			for btn := 0; btn < elevator.N_BUTTONS; btn++ {
				if currentElevatorSystem.HallRequests[floor][btn] != updatedElevatorSystem.HallRequests[floor][btn] {
					buttonIsChanged = true
					updatedButton.Button = elevio.ButtonType(btn)
					updatedButton.Floor = floor
				}
			}
		}
		if buttonIsChanged {
			OnRequestButtonPress(id, updatedElevatorSystem, updatedButton)
		}
	}

}

func OnRequestButtonPress(id string, elevatorSystem hra.ElevatorSystem, buttonEvent elevio.ButtonEvent) hra.ElevatorSystem {
	modifiedElevatorState := elevatorSystem.ElevatorStates[id]

	switch modifiedElevatorState.Behaviour {
	case elevator.EB_DoorOpen:
		if requests.ShouldClearImmediately(modifiedElevatorState, buttonEvent.Floor, buttonEvent.Button) {
			elevatorSystem.HallRequests[buttonEvent.Floor][buttonEvent.Button] = false
			log.Println("Clearing order!")
			timer.Start()
		}
	case elevator.EB_Idle:
		pair := requests.ChooseDirection(modifiedElevatorState, elevatorSystem.HallRequests)

		modifiedElevatorState.Direction = pair.Dir // Change to Direction instead of Dir
		modifiedElevatorState.Behaviour = pair.Behaviour

		switch pair.Behaviour {
		case elevator.EB_DoorOpen:
			elevio.SetDoorOpenLamp(true)
			timer.Start()
			elevatorSystem.HallRequests[buttonEvent.Floor][buttonEvent.Button] = false
			log.Println("Clearing order!")
		case elevator.EB_Moving:
			elevio.SetMotorDirection(modifiedElevatorState.Direction)
		}
	}
	setAllLights(elevatorSystem.HallRequests)
	elevatorSystem.ElevatorStates[id] = modifiedElevatorState
	return elevatorSystem

}

func OnFloorArrival(id string, elevatorSystem hra.ElevatorSystem) hra.ElevatorSystem {
	modifiedElevatorState := elevatorSystem.ElevatorStates[id]
	switch modifiedElevatorState.Behaviour {
	case elevator.EB_Moving:
		if requests.ShouldStop(modifiedElevatorState, elevatorSystem.HallRequests) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			modifiedElevatorState = requests.ClearAtCurrentFloor(modifiedElevatorState, elevatorSystem.HallRequests)
			setAllLights(elevatorSystem.HallRequests)
			elevio.SetDoorOpenLamp(true)
			timer.Start()
			modifiedElevatorState.Behaviour = elevator.EB_DoorOpen
		}
	}
	elevatorSystem.ElevatorStates[id] = modifiedElevatorState
	return elevatorSystem
}

func OnDoorTimeOut(id string, elevatorSystem hra.ElevatorSystem, hr hra.HallRequestsType) hra.LocalElevatorState {
	modifiedElevatorState := elevatorSystem.ElevatorStates[id]
	switch modifiedElevatorState.Behaviour {
	case elevator.EB_DoorOpen:
		pair := requests.ChooseDirection(modifiedElevatorState, hr)
		modifiedElevatorState.Direction = pair.Dir
		modifiedElevatorState.Behaviour = pair.Behaviour

		switch e.Behaviour {
		case elevator.EB_DoorOpen:
			timer.Start()
			modifiedElevatorState = requests.ClearAtCurrentFloor(modifiedElevatorState, hr)
			setAllLights(hr)
		case elevator.EB_Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(modifiedElevatorState.Direction)
		case elevator.EB_Moving:
			elevio.SetDoorOpenLamp(false)
		}
	}
	return e
}
