package timer

import (
	"time"
)

var (
	timerEndTime int64
	timerActive  bool
)

func TimerStart(duration int64) {
	timerEndTime = time.Now().UnixMicro() + duration
	timerActive = true
}

func TimerStop() {
	timerActive = false
}

func TimerTimedOut() bool {
	return (timerActive && time.Now().UnixMicro() > timerEndTime)
}
