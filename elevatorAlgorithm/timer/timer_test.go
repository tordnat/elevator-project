package timer

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/timer"
	"testing"
	"time"
)

func TestTimer(t *testing.T) {
	timer.Initialize()
	timer.Start()

	select {
	case <-timer.TimerChan:
	case <-time.After((elevator.DOOR_OPEN_DURATION_S + 1) * time.Second):
		t.Error("Failed to receive timer within expected time")
	}

	timer.Start()
	select {
	case <-timer.TimerChan:
	case <-time.After((elevator.DOOR_OPEN_DURATION_S + 1) * time.Second):
		t.Error("Failed to receive timer within expected time after reset")
	}

}
