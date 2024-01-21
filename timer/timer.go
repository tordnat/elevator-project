package timer

import (
	"time"
)

var (
	timerEndTime int64
	timerActive  bool
)

func Timer_start(duration int64) {
	timerEndTime = time.Now().UnixMicro() + duration
	timerActive = true
}

func Timer_stop() {
	timerActive = false
}

func Timer_timedOut() bool {
	return (timerActive && time.Now().UnixMicro() > timerEndTime)
}
