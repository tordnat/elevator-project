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
	"os"
)

const bcastPort int = 25565

func main() {
	// elevatorID := "0"
	log.Println("Elevator starting 🛗")
	elevatorPort, elevatorId := cmd.InitCommandLineArgs(os.Args)
	elevio.Init(fmt.Sprintf("localhost:%d", elevatorPort), elevator.N_FLOORS)

	buttonEvent := make(chan elevio.ButtonEvent)
	floorEvent := make(chan int)
	obstructionEvent := make(chan bool)
	elevio.SetDoorOpenLamp(false)
	go elevio.PollButtons(buttonEvent)
	go elevio.PollFloorSensor(floorEvent)
	go elevio.PollObstructionSwitch(obstructionEvent)

	//All channels of FSM go here
	orderAssignment := make(chan [][]bool)
	orderCompleted := make(chan requests.ClearFloorOrders)
	elevStateFromFSM := make(chan elevator.ElevatorState)

	//To not start in an obstructed state
	for elevio.GetObstruction() {
		elevio.SetDoorOpenLamp(true)
	}
	elevio.SetDoorOpenLamp(false)

	go fsm.FSM(orderAssignment, orderCompleted, floorEvent, obstructionEvent, elevStateFromFSM)
	requestSync.Sync(elevStateFromFSM, elevatorId, orderAssignment, orderCompleted)
}
