package timer

import "time"

func InitTimers() (*time.Timer, *time.Timer, *time.Timer) {
	doorTimer := time.NewTimer(0 * time.Second)
	obstructionTimer := time.NewTimer(0 * time.Second)
	inactivityTimer := time.NewTimer(0 * time.Second)
	<-doorTimer.C // Drain channels
	<-obstructionTimer.C
	<-inactivityTimer.C
	return doorTimer, obstructionTimer, inactivityTimer
}
