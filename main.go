package main

import (
	"elevator-project/timer"
	"elevatorDriver/elevio"
	"fmt"
	"networkDriver/bcast"
)

func main() {
	elevio.Init("localhost:15657", 4)
	elevio.SetMotorDirection(elevio.MD_Down)

	peerTxEnable := make(chan string)
	go bcast.Transmitter(15647, peerTxEnable)
	peerTxEnable <- "true"
	fmt.Printf("💩 is running\n")
	timer.Timer_start(10000000)
	for {
		if timer.Timer_timedOut() {
			fmt.Println("Timer stopped")
			return
		}
	}

}
