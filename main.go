package main

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/fsm"
	"elevatorAlgorithm/hra"
	"elevatorAlgorithm/timer"
	"elevatorDriver/elevio"
	"fmt"
	"log"
	"networkDriver/bcast"
	"time"
)

const bcastPort int = 25565

func main() {
	// elevatorID := "0"
	log.Println("Elevator starting 🛗")
	elevio.Init("localhost:15657", elevator.N_FLOORS)

	timer.Initialize()
	buttonEvent := make(chan elevio.ButtonEvent)
	floorEvent := make(chan int)
	obstructionEvent := make(chan bool)
	networkReciever := make(chan hra.ElevatorSystem)
	networkTransmitter := make(chan hra.ElevatorSystem)
	elevio.SetDoorOpenLamp(false)
	go elevio.PollButtons(buttonEvent)
	go elevio.PollFloorSensor(floorEvent)
	go elevio.PollObstructionSwitch(obstructionEvent)
	go bcast.Receiver(bcastPort, networkReciever)
	go bcast.Transmitter(bcastPort, networkTransmitter)

	//All channels of FSM go here
	orderAssignment := make(chan elevator.Order)      // This type is perhaps a bit weirdly defined. Events to this channel should be generated from HRA assignments.
	orderCompleted := make(chan fsm.ClearFloorOrders) //TODO: handle completed order deletion. Maybe a channel is overkill

	go fsm.FSM(orderAssignment, orderCompleted, floorEvent, obstructionEvent) //maybe add timer also?
	//go orderEventGenerator(orderAssignment, networkReciever, elevatorID)
	go removeOrder(orderCompleted)
	go test(orderAssignment)
	for {
	}
}

// Clear the order and send to network. If it was succesfully cleared we should quickly get back a new output from HRA where the order i cleared. If it's not cleared within door has closed, something went wrong, we should disconnect from network? Restart?
func removeOrder(orderCompletionRequestTVchannel chan fsm.ClearFloorOrders) {
	for order := range orderCompletionRequestTVchannel {
		log.Println("Order my ballz", order)
	}
}

func test(orderAssignment chan elevator.Order) {
	time.Sleep(5 * time.Second)
	log.Println("Sending assignment")
	orderAssignment <- elevator.Order{Floor: 1, Button: elevio.BT_HallDown}
	log.Println("Sent assignment")
	time.Sleep(5 * time.Second)
	orderAssignment <- elevator.Order{Floor: 3, Button: elevio.BT_HallDown}
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
