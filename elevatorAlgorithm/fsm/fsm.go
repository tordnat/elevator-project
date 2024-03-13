package fsm

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/requests"
	"elevatorAlgorithm/timer"
	"elevatorDriver/elevio"
	"log"
	"networkDriver/peers"
	"os"
	"time"
)

const INACTIVITY_TIMEOUT = 9 * time.Second

func FSM(orderAssignment chan [][]bool, clearOrders chan requests.ClearFloorOrders, floorEvent chan int, obstructionEvent chan bool, elevStateToSync chan elevator.ElevatorState, peersReceiver chan peers.PeerUpdate) {
	elevState := elevator.ElevatorState{elevator.EB_Idle, elevio.GetFloor(), elevio.MD_Stop, [][]bool{
		{false, false, false},
		{false, false, false},
		{false, false, false},
		{false, false, false}}} //Fix this!

	doorTimer, obstructionTimer, inactivityTimer := timer.InitTimers()

	isObstructed := false
	var activePeers []string
	elevStateToSync <- elevState

	for {
		select {
		case orders := <-orderAssignment:
			elevState.Requests = orders
			elevState.Behaviour, elevState.Direction = updateOrders(elevState, doorTimer, inactivityTimer)
			clearOrders <- OrdersToClear(elevState)

		case floor := <-floorEvent:
			elevio.SetFloorIndicator(floor)
			elevState.Floor = floor
			var clearOrder requests.ClearFloorOrders
			elevState.Behaviour, clearOrder = OnFloorArrival(elevState, doorTimer, inactivityTimer, isObstructed, obstructionTimer)
			clearOrders <- clearOrder

		case peersUpdate := <-peersReceiver:
			activePeers = peersUpdate.Peers

		case <-doorTimer.C:
			if isObstructed {
				break
			}
			elevState.Behaviour, elevState.Direction = OnDoorTimeOut(elevState, doorTimer, inactivityTimer)
			var clearOrder requests.ClearFloorOrders
			clearOrder.Floor = elevState.Floor
			clearOrder.Cab = true
			clearOrder.HallUp = requests.ShouldClearHallUp(elevState.Floor, elevState.Direction, elevState.Requests)
			clearOrder.HallDown = requests.ShouldClearHallDown(elevState.Floor, elevState.Direction, elevState.Requests)
			clearOrders <- clearOrder

		case isObstructed = <-obstructionEvent:
			if elevState.Behaviour == elevator.EB_DoorOpen {
				resetElevatorTimers(isObstructed, obstructionTimer, doorTimer)
			}
		case <-obstructionTimer.C:
			if len(activePeers) > 1 { //If we are alone, we cannot reset on obstruction or inactivity without loosing orders
				os.Exit(1)
			}
		case <-inactivityTimer.C:
			if len(activePeers) > 1 {
				os.Exit(2)
			}
		}
		elevStateToSync <- elevState //Should this be in a timer?
	}
}

func resetElevatorTimers(isObstructed bool, obstructionTimer *time.Timer, doorTimer *time.Timer) { //Add doorTImerReset here as well
	if isObstructed {
		log.Println("Obstruction occured!")
		obstructionTimer.Reset(INACTIVITY_TIMEOUT)
	} else {
		obstructionTimer.Stop()
	}
	doorTimer.Reset(elevator.DOOR_OPEN_DURATION)
}

func updateOrders(elevState elevator.ElevatorState, doorTimer *time.Timer, inactivityTimer *time.Timer) (elevator.ElevatorBehaviour, elevio.MotorDirection) {
	if requests.HaveOrders(elevState.Floor, elevState.Requests) { //Consider movign if statement out of function
		switch elevState.Behaviour {
		case elevator.EB_Idle:
			pair := requests.ChooseDirection(elevState.Direction, elevState.Floor, elevState.Requests)

			elevState.Direction = pair.Dir
			elevState.Behaviour = pair.Behaviour

			switch pair.Behaviour {
			case elevator.EB_DoorOpen:
				elevio.SetDoorOpenLamp(true)
				doorTimer.Reset(elevator.DOOR_OPEN_DURATION)
			case elevator.EB_Moving:
				inactivityTimer.Reset(INACTIVITY_TIMEOUT)
				elevio.SetMotorDirection(elevState.Direction)
			}
		}
	}
	return elevState.Behaviour, elevState.Direction
}

func OrdersToClear(elevState elevator.ElevatorState) requests.ClearFloorOrders {
	emptyClear := requests.ClearFloorOrders{elevState.Floor, false, false, false}
	if elevState.Behaviour == elevator.EB_DoorOpen {
		orderToClearImmediately := requests.ClearAtFloor(elevState.Floor, elevState.Direction, elevState.Requests)
		if orderToClearImmediately != emptyClear {
			return orderToClearImmediately
		}
	}
	return emptyClear
}

func OnFloorArrival(elevState elevator.ElevatorState, doorTimer *time.Timer, inactivityTimer *time.Timer, isObstructed bool, obstructionTimer *time.Timer) (elevator.ElevatorBehaviour, requests.ClearFloorOrders) {
	clearOrder := requests.ClearFloorOrders{elevState.Floor, false, false, false}
	switch elevState.Behaviour {
	case elevator.EB_Moving:
		if requests.ShouldStop(elevState.Direction, elevState.Floor, elevState.Requests) {
			inactivityTimer.Stop()
			elevio.SetMotorDirection(elevio.MD_Stop)
			clearOrder.Floor = elevState.Floor
			clearOrder.Cab = true
			clearOrder.HallUp = requests.ShouldClearHallUp(elevState.Floor, elevState.Direction, elevState.Requests)
			clearOrder.HallDown = requests.ShouldClearHallDown(elevState.Floor, elevState.Direction, elevState.Requests)

			resetElevatorTimers(isObstructed, obstructionTimer, doorTimer)

			elevState.Behaviour = elevator.EB_DoorOpen
			elevio.SetDoorOpenLamp(true)

		}
	}
	return elevState.Behaviour, clearOrder
}

func OnDoorTimeOut(elevState elevator.ElevatorState, doorTimer *time.Timer, inactivityTimer *time.Timer) (elevator.ElevatorBehaviour, elevio.MotorDirection) {
	switch elevState.Behaviour {
	case elevator.EB_DoorOpen:
		pair := requests.ChooseDirection(elevState.Direction, elevState.Floor, elevState.Requests)
		elevState.Direction = pair.Dir
		elevState.Behaviour = pair.Behaviour

		switch elevState.Behaviour {
		case elevator.EB_DoorOpen:
			doorTimer.Reset(elevator.DOOR_OPEN_DURATION)
		case elevator.EB_Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevState.Direction)
		case elevator.EB_Moving:
			inactivityTimer.Reset(INACTIVITY_TIMEOUT)
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevState.Direction)
		}
	}
	return elevState.Behaviour, elevState.Direction
}
