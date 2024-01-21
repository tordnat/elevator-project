package timer

import (
    "time"
)

var(
	timerEndTime int64;
	timerActive bool;
)

func timer_start(duration int64) {
	timerEndTime = time.Now().UnixMicro() + duration;
	timerActive = true;
}

func timer_stop() {
	timerActive = false;
}

func timer_timedOut() bool{
	return (timerActive && time.Now().UnixMicro() > timerEndTime);
}