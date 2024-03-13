package main

import (
	"elevator-project/cmd"
	"elevator-project/requestSync"
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/fsm"
	"elevatorAlgorithm/requests"
	"elevatorDriver/elevio"
	"fmt"
	"log"
	"networkDriver/peers"
	"os"
)

const peersPort int = 25566

func main() {
	// elevatorID := "0"
	log.Println("Elevator starting 🛗")
	elevatorPort, elevatorId := cmd.InitCommandLineArgs(os.Args)
	elevio.Init(fmt.Sprintf("localhost:%d", elevatorPort), elevator.N_FLOORS)
	buttonEvent := make(chan elevio.ButtonEvent)
	floorEvent := make(chan int)
	obstructionEvent := make(chan bool)

	peersTransmitter := make(chan bool)
	peersReceiverFSM := make(chan peers.PeerUpdate)
	peersReceiverRequestSync := make(chan peers.PeerUpdate)

	//Initialize to defined state
	elevio.SetMotorDirection(elevio.MD_Down)
	for elevio.GetFloor() == -1 {
	}
	elevio.SetMotorDirection(elevio.MD_Stop)
	for elevio.GetObstruction() {
		elevio.SetDoorOpenLamp(true)
	}
	elevio.SetDoorOpenLamp(false)
	log.Println("Elevator", elevatorId, "initialized")

	go peers.Transmitter(peersPort, elevatorId, peersTransmitter)
	go peers.ReceiverForwarder(peersPort, []chan peers.PeerUpdate{peersReceiverFSM, peersReceiverRequestSync})

	go elevio.PollButtons(buttonEvent)
	go elevio.PollFloorSensor(floorEvent)
	go elevio.PollObstructionSwitch(obstructionEvent)

	//All channels of FSM go here
	orderAssignment := make(chan [][]bool)
	orderCompleted := make(chan requests.ClearFloorOrders)
	elevStateFromFSM := make(chan elevator.Elevator)

	go fsm.FSM(orderAssignment, orderCompleted, floorEvent, obstructionEvent, elevStateFromFSM, peersReceiverFSM)
	requestSync.Sync(elevStateFromFSM, elevatorId, orderAssignment, orderCompleted, peersReceiverRequestSync)
}
