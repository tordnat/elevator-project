package timer

import (
	"elevatorAlgorithm/elevator"
	"time"
)

var (
	TimerActive bool
)

func Start() {
	timer := time.NewTimer(time.Second * elevator.DOOR_OPEN_DURATION_S)
	TimerActive = true
	go func() {
		select {
		case <-timer.C:
			TimerActive = false
		}
	}()
}
