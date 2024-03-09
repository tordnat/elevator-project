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

func FSM(orderAssignment chan [][]bool, clearOrders chan requests.ClearFloorOrders, floorEvent chan int, obstructionEvent chan bool, elevStateToSync chan elevator.ElevatorState) {
	elevState := elevator.ElevatorState{elevator.EB_Idle, -1, elevio.MD_Stop, [][]bool{
		{false, false, false},
		{false, false, false},
		{false, false, false},
		{false, false, false}}} //This 2d array should be properly filled

	log.Println("Initializing")
	elevState.Floor = elevio.GetFloor()
	if elevState.Floor == -1 {
		elevio.SetMotorDirection(elevio.MD_Down)
		elevState.Direction = elevio.MD_Down
		elevState.Behaviour = elevator.EB_Moving
	}
	for {
		select {
		case orderAssignments := <-orderAssignment:
			log.Println("New assignments ", orderAssignments)
			elevState.Requests = orderAssignments
			elevState.Behaviour, elevState.Direction = onNewAssignmentRequest(elevState)
			clearOrders <- newOrderAssignmentClear(elevState) // This must be run last to not clear orders at the wrong place

		case floor := <-floorEvent:
			elevState.Floor = floor
			var clearOrder requests.ClearFloorOrders
			elevState.Behaviour, clearOrder = OnFloorArrival(elevState)
			clearOrders <- clearOrder

		case <-timer.TimerChan:
			log.Println("Door timeout")
			timer.Timedout = true
			elevState.Behaviour, elevState.Direction = OnDoorTimeOut(elevState)
			if elevState.Behaviour == elevator.EB_DoorOpen {
				var clearOrder requests.ClearFloorOrders
				clearOrder.Floor = elevState.Floor
				clearOrder.Cab = true
				clearOrder.HallUp = requests.ShouldClearHallUp(elevState.Floor, elevState.Direction, elevState.Requests)
				clearOrder.HallDown = requests.ShouldClearHallDown(elevState.Floor, elevState.Direction, elevState.Requests)
				clearOrders <- requests.ClearFloorOrders{elevState.Floor, true, true, true}
			}

		case <-obstructionEvent:
			if elevState.Behaviour == elevator.EB_DoorOpen {
				timer.Start()
			}
		}
		elevStateToSync <- elevState //Must find out if this is a good place to sync elevState
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

func newOrderAssignmentClear(elevState elevator.ElevatorState) requests.ClearFloorOrders {
	emptyClear := requests.ClearFloorOrders{elevState.Floor, false, false, false}
	var clearOrder requests.ClearFloorOrders
	switch elevState.Behaviour {
	case elevator.EB_DoorOpen:
		orderToClearImmediately := requests.ClearAtFloor(elevState.Floor, elevState.Direction, elevState.Requests)
		if orderToClearImmediately != emptyClear {
			return orderToClearImmediately
		}
	case elevator.EB_Idle:
		pair := requests.ChooseDirection(elevState.Direction, elevState.Floor, elevState.Requests)

		elevState.Direction = pair.Dir
		elevState.Behaviour = pair.Behaviour

		switch pair.Behaviour {
		case elevator.EB_DoorOpen:
			clearOrder.Floor = elevState.Floor
			clearOrder.Cab = true
			clearOrder.HallUp = requests.ShouldClearHallUp(elevState.Floor, elevState.Direction, elevState.Requests)
			clearOrder.HallDown = requests.ShouldClearHallDown(elevState.Floor, elevState.Direction, elevState.Requests)
			return clearOrder
		}
	}
	return emptyClear
}

func onNewAssignmentRequest(elevState elevator.ElevatorState) (elevator.ElevatorBehaviour, elevio.MotorDirection) {
	if requests.HaveOrders(elevState.Floor, elevState.Requests) { //Consider movign if statement out of function
		switch elevState.Behaviour {
		case elevator.EB_DoorOpen:
			emptyClear := requests.ClearFloorOrders{elevState.Floor, false, false, false}
			ordersToClear := requests.ClearAtFloor(elevState.Floor, elevState.Direction, elevState.Requests)
			if ordersToClear != emptyClear {
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
			case elevator.EB_Moving:
				elevio.SetMotorDirection(elevState.Direction)
			}
		}
		setAllLights(elevState.Requests)
		return elevState.Behaviour, elevState.Direction
	} else {
		return elevator.EB_Idle, elevio.MD_Stop
	}
}

func OnFloorArrival(elevState elevator.ElevatorState) (elevator.ElevatorBehaviour, requests.ClearFloorOrders) {
	var clearOrder requests.ClearFloorOrders
	switch elevState.Behaviour {
	case elevator.EB_Moving:
		if requests.ShouldStop(elevState.Direction, elevState.Floor, elevState.Requests) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			clearOrder.Floor = elevState.Floor
			clearOrder.Cab = true
			clearOrder.HallUp = requests.ShouldClearHallUp(elevState.Floor, elevState.Direction, elevState.Requests)
			clearOrder.HallDown = requests.ShouldClearHallDown(elevState.Floor, elevState.Direction, elevState.Requests)

			setAllLights(elevState.Requests)
			elevio.SetDoorOpenLamp(true)
			timer.Start()
			elevState.Behaviour = elevator.EB_DoorOpen
		}
	}
	return elevState.Behaviour, clearOrder
}

func OnDoorTimeOut(elevState elevator.ElevatorState) (elevator.ElevatorBehaviour, elevio.MotorDirection) {
	switch elevState.Behaviour {
	case elevator.EB_DoorOpen:
		pair := requests.ChooseDirection(elevState.Direction, elevState.Floor, elevState.Requests)
		elevState.Direction = pair.Dir
		elevState.Behaviour = pair.Behaviour

		switch elevState.Behaviour {
		case elevator.EB_DoorOpen:
			timer.Start()
			setAllLights(elevState.Requests)
		case elevator.EB_Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevState.Direction)
		case elevator.EB_Moving:
			elevio.SetDoorOpenLamp(false)
		}
	}
	return elevState.Behaviour, elevState.Direction
}
