package main

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/fsm"
	"elevatorAlgorithm/hra"
	"elevatorAlgorithm/timer"
	"elevatorDriver/elevio"
	"log"
	"networkDriver/bcast"
)

const bcastPort int = 25565
const peersPort int = 25566

const (
	unknownOrder = iota
	noOrder
	unconfirmedOrder
	confirmedOrder
)

func main() {
	elevatorSystem := hra.ElevatorSystem{
		HallRequests: [][]int{
			{0, 0}, {0, 0}, {0, 0}, {0, 0},
		},
		ElevatorStates: map[string]hra.LocalElevatorState{
			"0": {
				Behaviour:   elevator.EB_Idle,
				Floor:       -1, //maybe not allowed
				Direction:   elevio.MD_Stop,
				CabRequests: []int{0, 0, 0, 0},
			},
		},
	}

	var confirmedOrders [][]bool
	elevatorID := "0"
	log.Println("Elevator starting 🛗")
	elevio.Init("localhost:15657", elevator.N_FLOORS)

	if elevio.GetFloor() == -1 {
		elevatorSystem.ElevatorStates[elevatorID] = fsm.OnInitBetweenFloors(elevatorSystem.ElevatorStates[elevatorID])
	}
	timer.Initialize()
	buttonEvent := make(chan elevio.ButtonEvent)
	floorEvent := make(chan int)
	networkReciever := make(chan hra.ElevatorSystem)
	networkTransmitter := make(chan hra.ElevatorSystem)
	elevio.SetDoorOpenLamp(false)
	go elevio.PollButtons(buttonEvent)
	go elevio.PollFloorSensor(floorEvent)
	go bcast.Receiver(bcastPort, networkReciever)
	go bcast.Transmitter(bcastPort, networkTransmitter)

	elevio.SetDoorOpenLamp(false)
	go elevio.PollButtons(buttonEvent)
	go elevio.PollFloorSensor(floorEvent)
	go bcast.Receiver(bcastPort, networkReciever)
	go bcast.Transmitter(bcastPort, networkTransmitter)

	for {
		log.Println("running")

		// Somehow remains stuck on init, was to sleeby to fix, sowi
		// - We need a deep copy of the local elevator system to then modify and compare it in the state transition
		// - either have to convert data structures, or define and use som deep copy for elevatorSystem
		select {
		case event := <-buttonEvent:
			log.Println("Button event ", event)
			updatedElevatorSystem := elevatorSystem
			updatedElevatorState := updatedElevatorSystem.ElevatorStates[elevatorID]
			if event.Button == elevio.BT_Cab {
				updatedElevatorState.CabRequests[event.Floor] = 1
			} else {
				updatedElevatorSystem.HallRequests[event.Floor][event.Button] = 1
			}
			updatedElevatorSystem.ElevatorStates[elevatorID] = updatedElevatorState
			encodedElsys := hra.Encode(updatedElevatorSystem)
			log.Println(updatedElevatorSystem)
			log.Println(encodedElsys)
			networkTransmitter <- updatedElevatorSystem
		case event := <-floorEvent:
			log.Println("Floor event", event)
			updatedElevatorSystem := elevatorSystem
			updatedElevatorState := elevatorSystem.ElevatorStates[elevatorID]
			updatedElevatorState.Floor = event
			updatedElevatorSystem.ElevatorStates[elevatorID] = updatedElevatorState
			networkTransmitter <- updatedElevatorSystem
			log.Println("Sending", event)

		case <-timer.TimerChan: // Should be able to remain separate since no new orders not known to the network exist in here....probably
			log.Println("Door timeout")
			updatedElevatorSystem := elevatorSystem
			timer.Timedout = true
			updatedElevatorState := fsm.OnDoorTimeOut(elevatorID, elevatorSystem, confirmedOrders)
			updatedElevatorSystem.ElevatorStates[elevatorID] = updatedElevatorState
			networkTransmitter <- updatedElevatorSystem
		case updatedElevatorSystem := <-networkReciever:
			updatedElevatorState := updatedElevatorSystem.ElevatorStates[elevatorID]
			for floor := 0; floor < elevator.N_FLOORS; floor++ {
				if updatedElevatorState.CabRequests[floor] != 0 {
					updatedElevatorState.CabRequests[floor] = confirmedOrder
				}
				for btn := 0; btn < elevator.N_HALL_BUTTONS; btn++ {
					if updatedElevatorSystem.HallRequests[floor][btn] != 0 {
						updatedElevatorSystem.HallRequests[floor][btn] = confirmedOrder
					}
				}
			}
			updatedElevatorSystem.ElevatorStates[elevatorID] = updatedElevatorState
			log.Println("Updated elevator system")
			encodedElsys := hra.Encode(updatedElevatorSystem)
			encodedCurrsys := hra.Encode(elevatorSystem)
			log.Println(encodedElsys)
			log.Println(encodedCurrsys)
			confirmedOrders = hra.Decode(hra.AssignRequests(encodedElsys))[elevatorID]
			log.Println(confirmedOrders)
			elevatorSystem = fsm.Transition(elevatorID, elevatorSystem, updatedElevatorSystem, confirmedOrders)
		}
	}
}
