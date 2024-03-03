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
	var elevatorSystem hra.ElevatorSystem
	elevatorID := "0"
	log.Println("Elevator starting 🛗")
	elevio.Init("localhost:15657", elevator.N_FLOORS)

	if elevio.GetFloor() == -1 {
		fsm.OnInitBetweenFloors()
	}
	timer.Initialize()
	buttonEvent := make(chan elevio.ButtonEvent)
	floorEvent := make(chan int)
	networkDummy := make(chan hra.ElevatorSystem)
	elevio.SetDoorOpenLamp(false)
	go elevio.PollButtons(buttonEvent)
	go elevio.PollFloorSensor(floorEvent)

	for {
		//få vekk timer, før go routine, fjerne hr/assigning, dette skal håndteres et annet sted
		// ++ include obstruction: should just reset timer for door
		select {
		case updatedElevatorSystem := <-networkDummy:
			log.Println("Updated elevator system")
		case event := <-buttonEvent:
			log.Println("Button event")
		case event := <-floorEvent:
			log.Println("Floor event")
		case <-timer.TimerChan:
			timer.Timedout = true
			log.Println("Got timer channel event")
		}
		time.Sleep(1 * time.Millisecond)
	}
}
