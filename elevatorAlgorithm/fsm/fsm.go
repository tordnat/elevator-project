package fsm

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/requests"
	"elevatorAlgorithm/timer"
	"elevatorDriver/elevio"
	"log"
)

type ClearFloorOrders struct {
	Floor    int
	HallUp   bool
	HallDown bool
	Cab      bool
}

// IMPORTANT: functions should either be completly pure,
// or they should communicate using channels and hardware.
// E.g execution of actions vs mutating data
// State transition functions may be the only exception, they return the new state. We could change the functions to be pure and return a list of HW actions, but a bit bloated?
// There is no need to use terminology like confirmed orders here. Alle orders are bools in the fsm, therefore they are simply orders.

func FSM(orderAssignment chan elevator.Order, clearOrders chan ClearFloorOrders, floorEvent chan int, obstructionEvent chan bool) {
	elevState := elevator.ElevatorState{elevator.EB_Idle, -1, elevio.MD_Stop, [][]bool{
		{false, false, false},
		{false, false, false},
		{false, false, false},
		{false, false, false}}} //This 2d array should be properly filled

	log.Println("Initializing")
	elevState.Floor = elevio.GetFloor()
	if elevState.Floor == -1 {
		elevState = OnInitBetweenFloors(elevState)
	}
	for {
		select {
		case event := <-orderAssignment: //Analogue of button press
			log.Println("New assignment ", event)
			clearOrders <- newReqClear(elevState, event)
			elevState.Behaviour, elevState.Direction = onNewRequest(elevState, event)

		case event := <-floorEvent:
			log.Println("At floor", event)
			elevState.Floor = event
			var clearOrder ClearFloorOrders
			elevState.Behaviour, clearOrder = OnFloorArrival(elevState)
			clearOrders <- clearOrder
			log.Println("Got past clearOrders in floor")

		case <-timer.TimerChan:
			log.Println("Door timeout")
			timer.Timedout = true
			elevState.Behaviour, elevState.Direction = OnDoorTimeOut(elevState)
			if elevState.Behaviour == elevator.EB_DoorOpen {
				var clearOrder ClearFloorOrders
				clearOrder.Floor = elevState.Floor
				clearOrder.Cab = true
				clearOrder.HallUp = requests.ShouldClearHallUp(elevState.Floor, elevState.Direction, elevState.Requests)
				clearOrder.HallDown = requests.ShouldClearHallDown(elevState.Floor, elevState.Direction, elevState.Requests)
				clearOrders <- ClearFloorOrders{elevState.Floor, true, true, true}
			}

		case <-obstructionEvent:
			if elevState.Behaviour == elevator.EB_DoorOpen {
				timer.Start()
			}
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

func newReqClear(elevState elevator.ElevatorState, orderEvent elevator.Order) ClearFloorOrders {
	var clearOrder ClearFloorOrders
	switch elevState.Behaviour {
	case elevator.EB_DoorOpen:
		if requests.ShouldClearImmediately(elevState.Floor, elevState.Direction, orderEvent) {
			clearOrder.Floor = orderEvent.Floor
			clearOrder.Cab = true
			return clearOrder
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
	return ClearFloorOrders{elevState.Floor, false, false, false}
}

func onNewRequest(elevState elevator.ElevatorState, orderEvent elevator.Order) (elevator.ElevatorBehaviour, elevio.MotorDirection) {
	switch elevState.Behaviour {
	case elevator.EB_DoorOpen:
		if requests.ShouldClearImmediately(elevState.Floor, elevState.Direction, orderEvent) {
			timer.Start()
		} else {
			elevState.Requests[orderEvent.Floor][orderEvent.Button] = true
		}
	case elevator.EB_Moving:
		elevState.Requests[orderEvent.Floor][orderEvent.Button] = true
	case elevator.EB_Idle:
		elevState.Requests[orderEvent.Floor][orderEvent.Button] = true
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
}

func OnFloorArrival(elevState elevator.ElevatorState) (elevator.ElevatorBehaviour, ClearFloorOrders) {
	var clearOrder ClearFloorOrders
	log.Println("On floor arrival")
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
