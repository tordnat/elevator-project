package fsm

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/requests"
	"elevatorDriver/elevio"
	"log"
	"os"
	"time"
)

func FSM(orderAssignment chan [][]bool, clearOrders chan requests.ClearFloorOrders, floorEvent chan int, obstructionEvent chan bool, elevStateToSync chan elevator.ElevatorState) {
	elevState := elevator.ElevatorState{elevator.EB_Idle, -1, elevio.MD_Stop, [][]bool{
		{false, false, false},
		{false, false, false},
		{false, false, false},
		{false, false, false}}}

	doorTimer := time.NewTimer(0 * time.Second)
	obstructedTimer := time.NewTimer(0 * time.Second)
	<-doorTimer.C       // Drain channel
	<-obstructedTimer.C // Drain channel
	log.Println("Initializing Elevator FSM")
	elevState = onInitBetweenFloors(elevState)
	for elevio.GetFloor() == -1 {

	}
	elevio.SetMotorDirection(elevio.MD_Stop)
	elevState.Direction = elevio.MD_Stop
	elevState.Behaviour = elevator.EB_Idle
	elevState.Floor = elevio.GetFloor()
	isObstructed := false
	elevStateToSync <- elevState
	for {
		select {
		case orders := <-orderAssignment:
			elevState.Requests = orders
			elevState.Behaviour, elevState.Direction = updateOrders(elevState, doorTimer)
			clearOrders <- OrdersToClear(elevState) // This must be run last to not clear orders at the wrong place

		case floor := <-floorEvent:
			elevState.Floor = floor
			var clearOrder requests.ClearFloorOrders
			elevState.Behaviour, clearOrder = OnFloorArrival(elevState, doorTimer)
			clearOrders <- clearOrder

		case <-doorTimer.C:
			if isObstructed {
				break
			}
			elevState.Behaviour, elevState.Direction = OnDoorTimeOut(elevState, doorTimer)
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
					obstructedTimer.Reset(time.Second * 7)
				} else {
					obstructedTimer.Stop()
				}
				doorTimer.Reset(elevator.DOOR_OPEN_DURATION_S * time.Second)
			}
		case <-obstructedTimer.C:
			os.Exit(1)
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

func updateOrders(elevState elevator.ElevatorState, doorTimer *time.Timer) (elevator.ElevatorBehaviour, elevio.MotorDirection) {
	if requests.HaveOrders(elevState.Floor, elevState.Requests) { //Consider movign if statement out of function
		switch elevState.Behaviour {
		case elevator.EB_Idle:
			pair := requests.ChooseDirection(elevState.Direction, elevState.Floor, elevState.Requests)

			elevState.Direction = pair.Dir
			elevState.Behaviour = pair.Behaviour

			switch pair.Behaviour {
			case elevator.EB_DoorOpen:
				elevio.SetDoorOpenLamp(true)
				doorTimer.Reset(elevator.DOOR_OPEN_DURATION_S * time.Second)
			case elevator.EB_Moving:
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

func OnFloorArrival(elevState elevator.ElevatorState, doorTimer *time.Timer) (elevator.ElevatorBehaviour, requests.ClearFloorOrders) {
	clearOrder := requests.ClearFloorOrders{elevState.Floor, false, false, false}
	switch elevState.Behaviour {
	case elevator.EB_Moving:
		if requests.ShouldStop(elevState.Direction, elevState.Floor, elevState.Requests) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			clearOrder.Floor = elevState.Floor
			clearOrder.Cab = true
			clearOrder.HallUp = requests.ShouldClearHallUp(elevState.Floor, elevState.Direction, elevState.Requests)
			clearOrder.HallDown = requests.ShouldClearHallDown(elevState.Floor, elevState.Direction, elevState.Requests)

			doorTimer.Reset(elevator.DOOR_OPEN_DURATION_S * time.Second)
			elevState.Behaviour = elevator.EB_DoorOpen
			elevio.SetDoorOpenLamp(true)
		}
	}
	return elevState.Behaviour, clearOrder
}

func OnDoorTimeOut(elevState elevator.ElevatorState, doorTimer *time.Timer) (elevator.ElevatorBehaviour, elevio.MotorDirection) {
	switch elevState.Behaviour {
	case elevator.EB_DoorOpen:
		pair := requests.ChooseDirection(elevState.Direction, elevState.Floor, elevState.Requests)
		elevState.Direction = pair.Dir
		elevState.Behaviour = pair.Behaviour

		switch elevState.Behaviour {
		case elevator.EB_DoorOpen:
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
