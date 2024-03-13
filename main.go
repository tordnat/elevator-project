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

	peersReciever := make(chan peers.PeerUpdate)
	peersTransmitter := make(chan bool)

	peersReceiverFSM := make(chan peers.PeerUpdate)
	peersReceiverRequestSync := make(chan peers.PeerUpdate)

	//Initialize to defined state
	elevio.SetMotorDirection(elevio.MD_Down)
	for elevio.GetFloor() == -1 {
	}
	for elevio.GetObstruction() {
		elevio.SetDoorOpenLamp(true)
	}
	elevio.SetDoorOpenLamp(false)
	elevio.SetMotorDirection(elevio.MD_Stop)

	go peers.Receiver(peersPort, peersReciever)
	go peers.Transmitter(peersPort, elevatorId, peersTransmitter)
	go peersChannelForwarder(peersReciever, []chan peers.PeerUpdate{peersReceiverFSM, peersReceiverRequestSync})

	go elevio.PollButtons(buttonEvent)
	go elevio.PollFloorSensor(floorEvent)
	go elevio.PollObstructionSwitch(obstructionEvent)

	//All channels of FSM go here
	orderAssignment := make(chan [][]bool)
	orderCompleted := make(chan requests.ClearFloorOrders)
	elevStateFromFSM := make(chan elevator.ElevatorState)

	go fsm.FSM(orderAssignment, orderCompleted, floorEvent, obstructionEvent, elevStateFromFSM, peersReceiverFSM)
	requestSync.Sync(elevStateFromFSM, elevatorId, orderAssignment, orderCompleted, peersReceiverRequestSync)
}

func peersChannelForwarder(sender chan peers.PeerUpdate, recipients []chan peers.PeerUpdate) {
	for peerUpdateMsg := range sender {
		for _, recipient := range recipients {
			go func(recipient chan peers.PeerUpdate, msg peers.PeerUpdate) { // Forwarding is not blocking
				recipient <- msg
			}(recipient, peerUpdateMsg)
		}
	}
}
