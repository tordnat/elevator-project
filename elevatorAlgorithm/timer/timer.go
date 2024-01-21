package timer

import (
	"time"
)

var (
	timerEndTime int64
	timerActive  bool
)

func Start(duration int64) {
	timerEndTime = time.Now().UnixMicro() + duration
	timerActive = true
}

func Stop() {
	timerActive = false
}

func TimedOut() bool {
	return (timerActive && time.Now().UnixMicro() > timerEndTime)
}
