package fsm

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/requests"
	"elevatorDriver/elevio"
	"log"
	"networkDriver/peers"
	"os"
	"time"
)

const INACTIVITY_TIMEOUT = 5 * time.Second

func FSM(orderAssignment chan [][]bool, clearOrders chan requests.ClearFloorOrders, floorEvent chan int, obstructionEvent chan bool, elevStateToSync chan elevator.ElevatorState, peersReceiver chan peers.PeerUpdate) {
	elevState := elevator.ElevatorState{elevator.EB_Idle, -1, elevio.MD_Stop, [][]bool{
		{false, false, false},
		{false, false, false},
		{false, false, false},
		{false, false, false}}}

	doorTimer := time.NewTimer(0 * time.Second)
	obstructionTimer := time.NewTimer(0 * time.Second)
	inactivityTimer := time.NewTimer(0 * time.Second)
	<-doorTimer.C // Drain channels
	<-obstructionTimer.C
	<-inactivityTimer.C
	log.Println("Initializing Elevator FSM")
	elevState = onInitBetweenFloors(elevState) // Initialization should not be inside FSM
	for elevio.GetFloor() == -1 {

	}
	elevio.SetMotorDirection(elevio.MD_Stop)
	elevState.Direction = elevio.MD_Stop
	elevState.Behaviour = elevator.EB_Idle
	elevState.Floor = elevio.GetFloor()
	isObstructed := false
	var activePeers []string
	elevStateToSync <- elevState
	for {
		select {
		case orders := <-orderAssignment:
			elevState.Requests = orders
			elevState.Behaviour, elevState.Direction = updateOrders(elevState, doorTimer, inactivityTimer)
			clearOrders <- OrdersToClear(elevState) // This must be run last to not clear orders at the wrong place

		case floor := <-floorEvent:
			elevio.SetFloorIndicator(floor)
			elevState.Floor = floor
			var clearOrder requests.ClearFloorOrders
			elevState.Behaviour, clearOrder = OnFloorArrival(elevState, doorTimer, inactivityTimer)
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
				if isObstructed {
					log.Println("Obstruction occured!")
					obstructionTimer.Reset(INACTIVITY_TIMEOUT)
				} else {
					obstructionTimer.Stop()
				}
				doorTimer.Reset(elevator.DOOR_OPEN_DURATION)
			}
		case <-obstructionTimer.C:
			if len(activePeers) > 1 {
				os.Exit(1)
			}
		case <-inactivityTimer.C:
			if len(activePeers) > 1 {
				os.Exit(2)
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

func OnFloorArrival(elevState elevator.ElevatorState, doorTimer *time.Timer, inactivityTimer *time.Timer) (elevator.ElevatorBehaviour, requests.ClearFloorOrders) {
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

			doorTimer.Reset(elevator.DOOR_OPEN_DURATION)
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
