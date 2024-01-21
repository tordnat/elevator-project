package main

import (
    "fmt"
    "time"
)

func getWallTime() float64 {
	type timeval struct {
		time_t      tv_sec;     /* seconds */
		suseconds_t tv_usec;    /* microseconds */
	}

	t := time.Now()
	return float64(t.unix()) + float64(t.Microsecond() * 0.000001)

}

var(
	timerEndTime float64;
	timerActive bool;
)

func timer_start(duration float64) Nil {
	timerEndTime := getWallTime() + duration;
	timerActive = True;
}

func timer_stop() Nil{
	timerActive = False;
}

func timer_timedOut() bool{
	return (timerActive && getWallTime() > timerEndTime);
}