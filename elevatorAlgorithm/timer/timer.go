package timer

import (
	"elevatorAlgorithm/elevator"
	"log"
	"time"
)

var DoorTimer *time.Timer

func Initialize() {
	DoorTimer = time.NewTimer(0 * time.Second)
	DoorTimer.Stop()
}

func Start() {
	log.Println("Started timer")
	if !DoorTimer.Stop() {
		<-DoorTimer.C // Drain the channel if the timer had already expired
	}
	DoorTimer.Reset(elevator.DOOR_OPEN_DURATION_S)
}

func PollTimer(timeChan chan time.Time) {
	timeChan <- <-DoorTimer.C
}
