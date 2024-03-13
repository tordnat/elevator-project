package timer

import "time"

type DoorTimer *time.Timer
type ObstructionTimer *time.Timer
type InacitivtyTimer *time.Timer

func InitTimers() (DoorTimer, ObstructionTimer, InacitivtyTimer) {
	doorTimer := time.NewTimer(0 * time.Second)
	obstructionTimer := time.NewTimer(0 * time.Second)
	inactivityTimer := time.NewTimer(0 * time.Second)
	<-doorTimer.C // Drain channels
	<-obstructionTimer.C
	<-inactivityTimer.C
	return doorTimer, obstructionTimer, inactivityTimer
}
