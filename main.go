package main

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/fsm"
	"elevatorAlgorithm/timer"
	"elevatorDriver/elevio"
	"log"
	"time"
)

func main() {

	log.Println("Elevator starting 🛗")
	elevio.Init("localhost:15657", elevator.N_FLOORS)
	if elevio.GetFloor() == -1 {
		fsm.OnInitBetweenFloors()
	}
	buttonEvent := make(chan elevio.ButtonEvent)
	floorEvent := make(chan int)
	doorCloseEvent := make(chan time.Time)
	timer.Initialize()
	elevio.SetDoorOpenLamp(false)
	go elevio.PollButtons(buttonEvent)
	go elevio.PollFloorSensor(floorEvent)
	go timer.PollTimer(doorCloseEvent)

	for {
		select {
		case event := <-buttonEvent:
			log.Println("Button event")
			fsm.OnRequestButtonPress(event.Floor, event.Button)
		case event := <-floorEvent:
			log.Println("Floor event")
			fsm.OnFloorArrival(event)
		case <-doorCloseEvent:
			log.Println("Door close event")
			fsm.OnDoorTimeOut()
		}
	}
}
