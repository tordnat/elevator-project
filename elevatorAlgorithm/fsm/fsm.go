package fsm

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/requests"
	"elevatorDriver/elevio"
	"fmt"
	"log"
	"time"
)

// IMPORTANT: functions should either be completly pure,
// or they should communicate using channels and hardware.
// E.g execution of actions vs mutating data
// State transition functions may be the only exception, they return the new state. We could change the functions to be pure and return a list of HW actions, but a bit bloated?
// There is no need to use terminology like confirmed orders here. Alle orders are bools in the fsm, therefore they are simply orders.
var doorTimer *time.Timer

func FSM(orderAssignment chan [][]bool, clearOrders chan requests.ClearFloorOrders, floorEvent chan int, obstructionEvent chan bool, elevStateToSync chan elevator.ElevatorState) {
	elevState := elevator.ElevatorState{elevator.EB_Idle, -1, elevio.MD_Stop, [][]bool{
		{false, false, false},
		{false, false, false},
		{false, false, false},
		{false, false, false}}} //This 2d array should be properly filled

	doorTimer = time.NewTimer(0 * time.Second)
	<-doorTimer.C // Drain channel

	log.Println("Initializing Elevator FSM")
	elevState.Floor = elevio.GetFloor()
	if elevState.Floor == -1 {
		elevState = onInitBetweenFloors(elevState)
	}

	for {
		select {
		case orderAssignments := <-orderAssignment:
			elevState.Requests = orderAssignments
			log.Println(elevState.Requests)
			log.Println(elevState.Behaviour)
			elevState.Behaviour, elevState.Direction = updateAssignmentRequest(elevState)
			setAllLights(elevState.Requests)

			clearOrders <- newOrderAssignmentClear(elevState) // This must be run last to not clear orders at the wrong place

		case floor := <-floorEvent:
			elevState.Floor = floor
			var clearOrder requests.ClearFloorOrders
			elevState.Behaviour, clearOrder = OnFloorArrival(elevState)
			clearOrders <- clearOrder

		case <-doorTimer.C:
			log.Println("Door timeout")
			elevState.Behaviour, elevState.Direction = OnDoorTimeOut(elevState)
			var clearOrder requests.ClearFloorOrders
			clearOrder.Floor = elevState.Floor
			clearOrder.Cab = true
			clearOrder.HallUp = requests.ShouldClearHallUp(elevState.Floor,

				elevState.Direction, elevState.Requests,
			)
			clearOrder.HallDown = requests.ShouldClearHallDown(elevState.Floor, elevState.Direction, elevState.Requests)
			fmt.Println("Cleared order ", clearOrder, "after timeout")
			clearOrders <- clearOrder

		case <-obstructionEvent:
			if elevState.Behaviour == elevator.EB_DoorOpen {
				doorTimer.Reset(elevator.DOOR_OPEN_DURATION_S * time.Second)
				log.Println("Resetting timer, obstructed")
			}
		}
		elevStateToSync <- elevState //Must find out if this is a good place to sync elevState
	}
}

func onInitBetweenFloors(e elevator.ElevatorState) elevator.ElevatorState {
	elevio.SetMotorDirection(elevio.MD_Down)
	e.Direction = elevio.MD_Down
	e.Behaviour = elevator.EB_Moving
	return e
}

func updateAssignmentRequest(elevState elevator.ElevatorState) (elevator.ElevatorBehaviour, elevio.MotorDirection) {
	if requests.HaveOrders(elevState.Floor, elevState.Requests) { //Consider movign if statement out of function
		switch elevState.Behaviour {
		case elevator.EB_Idle:
			pair := requests.ChooseDirection(elevState.Direction, elevState.Floor, elevState.Requests)

			elevState.Direction = pair.Dir
			elevState.Behaviour = pair.Behaviour

			switch pair.Behaviour {
			case elevator.EB_DoorOpen:
				elevio.SetDoorOpenLamp(true)
				log.Println("Resetting timer onNewAssignmentRequest from IDLE")
				doorTimer.Reset(elevator.DOOR_OPEN_DURATION_S * time.Second)
			case elevator.EB_Moving:
				elevio.SetMotorDirection(elevState.Direction)
			}
		}
	}
	return elevState.Behaviour, elevState.Direction
}

func newOrderAssignmentClear(elevState elevator.ElevatorState) requests.ClearFloorOrders {
	emptyClear := requests.ClearFloorOrders{elevState.Floor, false, false, false}
	if elevState.Behaviour == elevator.EB_DoorOpen {
		orderToClearImmediately := requests.ClearAtFloor(elevState.Floor, elevState.Direction, elevState.Requests)
		if orderToClearImmediately != emptyClear {
			return orderToClearImmediately
		}
	}
	return emptyClear
}

func OnFloorArrival(elevState elevator.ElevatorState) (elevator.ElevatorBehaviour, requests.ClearFloorOrders) {
	clearOrder := requests.ClearFloorOrders{elevState.Floor, false, false, false}
	switch elevState.Behaviour {
	case elevator.EB_Moving:
		if requests.ShouldStop(elevState.Direction, elevState.Floor, elevState.Requests) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			clearOrder.Floor = elevState.Floor
			clearOrder.Cab = true
			clearOrder.HallUp = requests.ShouldClearHallUp(elevState.Floor, elevState.Direction, elevState.Requests)
			clearOrder.HallDown = requests.ShouldClearHallDown(elevState.Floor, elevState.Direction, elevState.Requests)

			log.Println("Resetting timer")
			doorTimer.Reset(elevator.DOOR_OPEN_DURATION_S * time.Second)
			elevState.Behaviour = elevator.EB_DoorOpen
			elevio.SetDoorOpenLamp(true)
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
			log.Println("Resetting timer in OnDoorTimeout")
			doorTimer.Reset(elevator.DOOR_OPEN_DURATION_S * time.Second)
		case elevator.EB_Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevState.Direction)
		case elevator.EB_Moving:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevState.Direction)
		}
	}
	return elevState.Behaviour, elevState.Direction
}

func setAllLights(confirmedOrders [][]bool) {
	for floor := 0; floor < elevator.N_FLOORS; floor++ {
		for btn := 0; btn < elevator.N_BUTTONS; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, confirmedOrders[floor][btn])
		}
	}
}
