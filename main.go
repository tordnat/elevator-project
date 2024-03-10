package main

import (
	"elevator-project/cmd"
	"elevator-project/requestSync"
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/fsm"
	"elevatorAlgorithm/hra"
	"elevatorAlgorithm/requests"
	"elevatorDriver/elevio"
	"fmt"
	"log"
	"os"
	"time"
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
	orderAssignment := make(chan [][]bool, 1) //Buffered to prevent deadock between assugnemnt and clearing
	orderCompleted := make(chan requests.ClearFloorOrders)
	elevStateFromFSM := make(chan elevator.ElevatorState)

	go fsm.FSM(orderAssignment, orderCompleted, floorEvent, obstructionEvent, elevStateFromFSM)
	go requestSync.Sync(elevStateFromFSM, elevatorId, orderAssignment, orderCompleted)
	for {
	}
}

// Clear the order and send to network. If it was succesfully cleared we should quickly get back a new output from HRA where the order i cleared. If it's not cleared within door has closed, something went wrong, we should disconnect from network? Restart?
func removeOrder(orderCompletionRequestTVchannel chan requests.ClearFloorOrders) {
	for order := range orderCompletionRequestTVchannel {
		log.Println("New order", order)
	}
}

func addOrder(orderAssignment chan [][]bool) {
	time.Sleep(5 * time.Second)
	log.Println("Sending assignment")
	orderAssignment <- [][]bool{
		{true, false, false},
		{false, false, false},
		{false, false, false},
		{false, false, false}}
	log.Println("Sent assignment")
	time.Sleep(8 * time.Second)
	orderAssignment <- [][]bool{
		{false, false, false},
		{false, false, false},
		{false, false, false},
		{true, false, false}}
	time.Sleep(8 * time.Second)
	orderAssignment <- [][]bool{
		{false, false, false},
		{false, false, false},
		{false, false, false},
		{true, false, false}}
}

// TODO: Make this function generate orderAssignement events based on HRA output. Bascially it only sends an assignment if it's new, like a button press
// This function is not really a final implementation, but something that is easy to take appart to merge with other parts of the system
// Also: A better implementation is to not base the FSM on only getting a single new order. Simple fix now is to just send requests successively.
// Main problem is that we have no way of clearing orders from the FSM without it clearing them itself. This is why we need clearing on a sperate channel
// We need to store hra.ElevatorSystem in a plae where this function can access it. Only requestSync will have access to it currently
func orderEventGenerator(orderAssignment chan elevator.Order, networkReciever chan hra.ElevatorSystem, elevatorID string) {
	var oldAssignments [][]bool
	for networkMsg := range networkReciever {
		newAssignments, ok := hra.Decode(hra.AssignRequests(hra.Encode(networkMsg)))[elevatorID]
		if !ok {
			fmt.Println("Error")
		}
		i, j := findDifference(newAssignments, oldAssignments)
		if !(i == -1 || j == -1) { //Change this to a while loop
			orderAssignment <- elevator.Order{i, elevio.ButtonType(j)} //i and j must be double checked here.
		}
	}
}

func findDifference(a, b [][]bool) (int, int) {
	for i := range a {
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return i, j
			}
		}
	}
	return -1, -1 //Arrays were equal
}
