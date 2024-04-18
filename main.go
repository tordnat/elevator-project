package main

import (
	"elevator-project/cmd"
	"elevator-project/orderSync"
	"elevatorControl/elevator"
	"elevatorControl/fsm"
	"elevatorControl/orders"
	"elevatorDriver/elevio"
	"fmt"
	"log"
	"networkDriver/peers"
	"os"
)

const peersPort int = 25566

func main() {
	log.Println("Elevator is starting 🛗")
	elevatorPort, elevatorId := cmd.InitCommandLineArgs(os.Args)
	elevio.Init(fmt.Sprintf("localhost:%d", elevatorPort), elevator.N_FLOORS)
	buttonEvent := make(chan elevio.ButtonEvent)
	floorEvent := make(chan int)
	obstructionEvent := make(chan bool)

	peersTransmitter := make(chan bool)
	peersReceiverFSM := make(chan peers.PeerUpdate)
	peersReceiverorderSync := make(chan peers.PeerUpdate)

	orderAssignment := make(chan [][]bool)
	orderCompleted := make(chan orders.ClearFloorOrders)
	elevStateFromFSM := make(chan elevator.Elevator)

	//Initialize to defined state
	elevio.SetMotorDirection(elevio.MD_Down)
	for elevio.GetFloor() == -1 {
	}
	elevio.SetMotorDirection(elevio.MD_Stop)
	for elevio.GetObstruction() {
		elevio.SetDoorOpenLamp(true)
	}
	elevio.SetDoorOpenLamp(false)
	log.Println("Elevator", elevatorId, "initialized. Ready to join Network")

	go peers.Transmitter(peersPort, elevatorId, peersTransmitter)
	go peers.Reciever(peersPort, []chan peers.PeerUpdate{peersReceiverFSM, peersReceiverorderSync})

	go elevio.PollButtons(buttonEvent)
	go elevio.PollFloorSensor(floorEvent)
	go elevio.PollObstructionSwitch(obstructionEvent)

	go fsm.FSM(orderAssignment, orderCompleted, floorEvent, obstructionEvent, elevStateFromFSM, peersReceiverFSM)
	orderSync.Sync(elevStateFromFSM, elevatorId, orderAssignment, orderCompleted, peersReceiverorderSync)
}
