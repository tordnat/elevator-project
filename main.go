package main

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/fsm"
	"elevatorAlgorithm/hra"
	"elevatorAlgorithm/timer"
	"elevatorDriver/elevio"
	"log"
	"time"
)

func main() {
	elevatorSystem := hra.ElevatorSystem{
		HallRequests: [][]int{
			{0, 0}, {0, 0}, {0, 0}, {0, 0},
		},
		ElevatorStates: map[string]hra.LocalElevatorState{
			"0": {
				Behaviour:   elevator.EB_Moving,
				Floor:       2,
				Direction:   elevio.MD_Up,
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
	networkDummy := make(chan hra.ElevatorSystem)
	elevio.SetDoorOpenLamp(false)
	go elevio.PollButtons(buttonEvent)
	go elevio.PollFloorSensor(floorEvent)

	for {
		// Somehow remains stuck on init, was to sleeby to fix, sowi
		select {
		case event := <-buttonEvent:
			log.Println("Button event")
			updatedElevatorSystem := elevatorSystem
			updatedElevatorSystem.HallRequests[event.Floor][event.Button] = 1
			networkDummy <- updatedElevatorSystem
		case event := <-floorEvent:
			log.Println("Floor event")
			updatedElevatorSystem := elevatorSystem
			updatedElevatorState := elevatorSystem.ElevatorStates[elevatorID]
			updatedElevatorState.Floor = event
			updatedElevatorSystem.ElevatorStates[elevatorID] = updatedElevatorState
			networkDummy <- updatedElevatorSystem
		case <-timer.TimerChan: // Should be able to remain separate since no new orders not known to the network exist in here....probably
			timer.Timedout = true
			fsm.OnDoorTimeOut(elevatorID, elevatorSystem, confirmedOrders)
		case updatedElevatorSystem := <-networkDummy:
			// DO SYNC SHIT
			log.Println("Updated elevator system")
			encodedElsys := hra.Encode(updatedElevatorSystem)
			confirmedOrders = hra.Decode(hra.AssignRequests(encodedElsys))[elevatorID] // Shitty oneliner
			elevatorSystem = fsm.Transition(elevatorID, elevatorSystem, updatedElevatorSystem, confirmedOrders)
		}
		time.Sleep(1 * time.Millisecond)
	}
}
