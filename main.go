package main

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/fsm"
	"elevatorAlgorithm/timer"
	"elevatorDriver/elevio"
	"fmt"
	"networkDriver/bcast"
)

func main() {

	elevio.Init("localhost:15657", 4)
	elevio.SetMotorDirection(elevio.MD_Up)
	fmt.Println(elevator.N_FLOORS)
	peerTxEnable := make(chan string)
	go bcast.Transmitter(15647, peerTxEnable)
	if fsm.Fsm_lint_me {
		peerTxEnable <- "true"
	}
	fmt.Print("💩 is running\n")
	timer.TimerStart(10000000)
	for {
		if timer.TimerTimedOut() {
			fmt.Println("Timer stopped")
			return
		}
	}

}
