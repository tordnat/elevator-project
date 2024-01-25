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
	timer.Initialize()
	buttonEvent := make(chan elevio.ButtonEvent)
	floorEvent := make(chan int)
	elevio.SetDoorOpenLamp(false)
	go elevio.PollButtons(buttonEvent)
	go elevio.PollFloorSensor(floorEvent)

	for {
		select {
		case event := <-buttonEvent:
			log.Println("Button event")
			fsm.OnRequestButtonPress(event.Floor, event.Button)
		case event := <-floorEvent:
			log.Println("Floor event")
			fsm.OnFloorArrival(event)
		case <-timer.TimerChan:
			log.Println("Got timer channel event")
			fsm.OnDoorTimeOut()
		}
		time.Sleep(1 * time.Millisecond)
	}
}
