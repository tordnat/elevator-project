package timer

import (
	"elevatorAlgorithm/elevator"
	"sync"
	"time"
)

var (
	TimerChan  chan bool
	resetChan  chan struct{}
	timer      *time.Timer
	timerMutex sync.Mutex
	Timedout   bool
)

func Initialize() {
	TimerChan = make(chan bool, 1)
	resetChan = make(chan struct{}, 1)
	timer = time.NewTimer(time.Second)
	timer.Stop()

	go func() {
		for {
			select {
			case <-resetChan:
				if !timer.Stop() {
					select {
					case <-TimerChan:
					default:
					}
				}
				timer.Reset(time.Second * elevator.DOOR_OPEN_DURATION_S)
			case <-timer.C:
				TimerChan <- true
			}

		}
	}()
}

func Start() {
	Timedout = false
	timerMutex.Lock()
	defer timerMutex.Unlock()
	resetChan <- struct{}{}
}
