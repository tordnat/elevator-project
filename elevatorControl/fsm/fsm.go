package fsm

import (
	"elevatorControl/elevator"
	"elevatorControl/orders"
	"elevatorControl/timer"
	"elevatorDriver/elevio"
	"log"
	"networkDriver/peers"
	"os"
	"time"
)

const obstructionTimeout = 5 * time.Second
const inactivityTimeout = 9 * time.Second

func FSM(orderAssignment chan [][]bool, clearOrders chan orders.ClearFloorOrders, floorEvent chan int, obstructionEvent chan bool, elevStateToSync chan elevator.Elevator, peersReceiver chan peers.PeerUpdate) {
	elevState := elevator.NewElevator(elevator.EB_Idle, elevio.GetFloor(), elevio.MD_Stop, elevator.N_FLOORS)
	doorTimer, obstructionTimer, inactivityTimer := timer.InitTimers()

	isObstructed := false
	var activePeers []string
	elevStateToSync <- elevState

	for {
		select {
		case orders := <-orderAssignment:
			elevState.Orders = orders
			elevState.Behaviour, elevState.Direction = updateOrders(elevState, doorTimer, inactivityTimer, obstructionTimer, isObstructed)
			clearOrders <- OrdersToClear(elevState)

		case floor := <-floorEvent:
			elevio.SetFloorIndicator(floor)
			elevState.Floor = floor
			var clearOrder orders.ClearFloorOrders
			elevState.Behaviour, clearOrder = OnFloorArrival(elevState, doorTimer, inactivityTimer, isObstructed, obstructionTimer)
			clearOrders <- clearOrder

		case peersUpdate := <-peersReceiver:
			activePeers = peersUpdate.Peers

		case <-doorTimer.C:
			if isObstructed {
				break
			}
			elevState.Behaviour, elevState.Direction = OnDoorTimeOut(elevState, doorTimer, inactivityTimer, obstructionTimer, isObstructed)

			var clearOrder orders.ClearFloorOrders
			clearOrder.Floor = elevState.Floor
			clearOrder.Cab = true
			clearOrder.HallUp = orders.ShouldClearHallUp(elevState.Floor, elevState.Direction, elevState.Orders)
			clearOrder.HallDown = orders.ShouldClearHallDown(elevState.Floor, elevState.Direction, elevState.Orders)

			clearOrders <- clearOrder

		case isObstructed = <-obstructionEvent:
			if elevState.Behaviour == elevator.EB_DoorOpen {
				resetElevatorTimers(isObstructed, obstructionTimer, doorTimer)
			}
		case <-obstructionTimer.C:
			if len(activePeers) > 1 { //If we are alone, we cannot reset on obstruction or inactivity without loosing orders
				os.Exit(1)
			} else {
				obstructionTimer.Reset(obstructionTimeout)
			}
		case <-inactivityTimer.C:
			if len(activePeers) > 1 {
				os.Exit(2)
			} else {
				inactivityTimer.Reset(inactivityTimeout)
			}
		}
		select {
		case elevStateToSync <- elevState: // Prevent blocking
		default:
		}
	}
}

func resetElevatorTimers(isObstructed bool, obstructionTimer *time.Timer, doorTimer *time.Timer) { //Add doorTImerReset here as well
	if isObstructed {
		log.Println("Obstruction occured!")
		obstructionTimer.Reset(obstructionTimeout)
	} else {
		obstructionTimer.Stop()
	}
	doorTimer.Reset(elevator.DOOR_OPEN_DURATION)
}

func updateOrders(elevState elevator.Elevator, doorTimer *time.Timer, inactivityTimer *time.Timer, obstructionTimer *time.Timer, isObstructed bool) (elevator.ElevatorBehaviour, elevio.MotorDirection) {
	if orders.HaveOrders(elevState.Floor, elevState.Orders) {
		switch elevState.Behaviour {
		case elevator.EB_Idle:
			pair := orders.ChooseDirection(elevState.Direction, elevState.Floor, elevState.Orders)

			elevState.Direction = pair.Dir
			elevState.Behaviour = pair.Behaviour

			switch pair.Behaviour {
			case elevator.EB_DoorOpen:
				elevio.SetDoorOpenLamp(true)
				resetElevatorTimers(isObstructed, obstructionTimer, doorTimer)
			case elevator.EB_Moving:
				inactivityTimer.Reset(inactivityTimeout)
				elevio.SetMotorDirection(elevState.Direction)
			}
		}
	}
	return elevState.Behaviour, elevState.Direction
}

func OrdersToClear(elevState elevator.Elevator) orders.ClearFloorOrders {
	emptyClear := orders.ClearFloorOrders{elevState.Floor, false, false, false}
	if elevState.Behaviour == elevator.EB_DoorOpen {
		orderToClearImmediately := orders.ClearAtFloor(elevState.Floor, elevState.Direction, elevState.Orders)
		if orderToClearImmediately != emptyClear {
			return orderToClearImmediately
		}
	}
	return emptyClear
}

func OnFloorArrival(elevState elevator.Elevator, doorTimer *time.Timer, inactivityTimer *time.Timer, isObstructed bool, obstructionTimer *time.Timer) (elevator.ElevatorBehaviour, orders.ClearFloorOrders) {
	clearOrder := orders.ClearFloorOrders{elevState.Floor, false, false, false}
	switch elevState.Behaviour {
	case elevator.EB_Moving:
		if orders.ShouldStop(elevState.Direction, elevState.Floor, elevState.Orders) {
			inactivityTimer.Stop()
			elevio.SetMotorDirection(elevio.MD_Stop)
			clearOrder.Floor = elevState.Floor
			clearOrder.Cab = true
			clearOrder.HallUp = orders.ShouldClearHallUp(elevState.Floor, elevState.Direction, elevState.Orders)
			clearOrder.HallDown = orders.ShouldClearHallDown(elevState.Floor, elevState.Direction, elevState.Orders)

			resetElevatorTimers(isObstructed, obstructionTimer, doorTimer)

			elevState.Behaviour = elevator.EB_DoorOpen
			elevio.SetDoorOpenLamp(true)

		}
	}
	return elevState.Behaviour, clearOrder
}

func OnDoorTimeOut(elevState elevator.Elevator, doorTimer *time.Timer, inactivityTimer *time.Timer, obstructionTimer *time.Timer, isObstructed bool) (elevator.ElevatorBehaviour, elevio.MotorDirection) {
	switch elevState.Behaviour {
	case elevator.EB_DoorOpen:
		pair := orders.ChooseDirection(elevState.Direction, elevState.Floor, elevState.Orders)
		elevState.Direction = pair.Dir
		elevState.Behaviour = pair.Behaviour

		switch elevState.Behaviour {
		case elevator.EB_DoorOpen:
			//inactivityTimer.Reset(inactivityTimeout)
			resetElevatorTimers(isObstructed, obstructionTimer, doorTimer)
			//doorTimer.Reset(elevator.DOOR_OPEN_DURATION)
		case elevator.EB_Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevState.Direction)
		case elevator.EB_Moving:
			inactivityTimer.Reset(inactivityTimeout)
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevState.Direction)
		}
	}
	return elevState.Behaviour, elevState.Direction
}
