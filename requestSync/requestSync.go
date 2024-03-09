package requestSync

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/hra"
	"elevatorAlgorithm/requests"
	"elevatorDriver/elevio"
	"fmt"
	"log"
	"networkDriver/bcast"
	"networkDriver/peers"
	"time"
)

type StateMsg struct {
	Id           string
	Counter      uint64 //Non-monotonic counter to only recieve newest data
	HallRequests hra.HallRequestsType
	Elevator     hra.LocalElevatorState
}

// Must be imported
const (
	unknownOrder = iota
	noOrder
	unconfirmedOrder
	confirmedOrder
	servicedOrder
)
const bcastPort int = 25565
const peersPort int = 25566

var elevatorSystems map[string]hra.ElevatorSystem = make(map[string]hra.ElevatorSystem) //Could have been closure, but easier as global. Maybe we can't make it a closure if peer list should modify it

func Sync(elevatorSystemFromFSM chan elevator.ElevatorState, elevatorId string, orderAssignment chan [][]bool, orderCompleted chan requests.ClearFloorOrders) {
	btnEvent := make(chan elevio.ButtonEvent)
	networkReciever := make(chan StateMsg)
	networkTransmitter := make(chan StateMsg)
	peersReciever := make(chan peers.PeerUpdate)
	peersTransmitter := make(chan bool)

	go bcast.Receiver(bcastPort, networkReciever)
	go bcast.Transmitter(bcastPort, networkTransmitter)
	go peers.Receiver(peersPort, peersReciever)
	go peers.Transmitter(peersPort, elevatorId, peersTransmitter)

	go elevio.PollButtons(btnEvent)

	timer := time.NewTimer(0)
	goToFloor0()
	var msgCounter uint64 = 0
	var latestPeerList []string

	var elevatorSystem hra.ElevatorSystem
	for {
		select {
		case btn := <-btnEvent: //Got order
			elevatorSystem = AddOrder(elevatorId, elevatorSystem, btn)
			printElevatorSystem(elevatorSystem)
			msgCounter += 1 //To prevent forgetting counter, this should perhaps be in a seperate function
			networkTransmitter <- StateMsg{elevatorId, msgCounter, elevatorSystem.HallRequests, elevatorSystem.ElevatorStates[elevatorId]}

		case networkMsg := <-networkReciever: //Got message
			if networkMsg.Counter <= msgCounter {
				msgCounter += 1
				break
			}
			msgCounter = networkMsg.Counter

			elevatorSystem = Transition(elevatorId, elevatorSystem, networkMsg)
			// TODO add HRA assignment of order here
			// orderAssignment <- orderWhichAreConfirmedAndFromHRA

		case peersUpdate := <-peersReciever:
			latestPeerList = peersUpdate.Peers //Here we should also update the elevatorSystem map. Important to take the (hall)orders of lost peers before removing it
			_ = latestPeerList
		case elevator := <-elevatorSystemFromFSM:
			_ = elevator

		case orderToClear := <-orderCompleted:
			//Transmit to network that we want to clear
			//Bascially run a transition on our elevator system after having assigned the order as completed
			_ = orderToClear
		case <-timer.C: //Timer reset, send new state update
			msgCounter += 1
			networkTransmitter <- StateMsg{elevatorId, msgCounter, elevatorSystem.HallRequests, elevatorSystem.ElevatorStates[elevatorId]}
			timer.Reset(time.Millisecond * 1000)
		}
	}
}
func goToFloor0() {
	elevio.SetMotorDirection(elevio.MD_Down)
	for {
		if elevio.GetFloor() == 0 {
			elevio.SetMotorDirection(elevio.MD_Stop)
			return
		}
	}
}

// Improvements to transition: We modify both localElevSystem and elevatorSystems if a new node joins, this should only need to be done once, e.g using a function.
func Transition(ourId string, localElevSystem hra.ElevatorSystem, networkMsg StateMsg) hra.ElevatorSystem { //This function should maybe maybe be moved to main, or parts of it moved

	var localCabRequests []int
	if localElevState, ok := localElevSystem.ElevatorStates[networkMsg.Id]; !ok {
		localCabRequests = localElevState.CabRequests
	} else {
		//Add the network elevator to the local list here (if it is in peer list?)
		log.Println("Elevator with id", networkMsg.Id, "not in local list") //This should be logged as info, should only happen at the start and if we loose an elevator
		localElevSystem.ElevatorStates[networkMsg.Id] = networkMsg.Elevator
		elevatorSystems[networkMsg.Id] = localElevSystem // This is bad, should only have a single point to add/delete elevator from state!
		return localElevSystem
	}
	networkCabRequests := networkMsg.Elevator.CabRequests

	//Transition cab requests. Might want to hide this ugliness (this is just because go maps are a bit weird)

	tmpNetElevatorState := localElevSystem.ElevatorStates[networkMsg.Id]
	tmpNetElevatorState.CabRequests = transitionCabRequests(localCabRequests, networkCabRequests)
	localElevSystem.ElevatorStates[networkMsg.Id] = tmpNetElevatorState

	localElevSystem.HallRequests = transitionHallRequests(localElevSystem.HallRequests, networkMsg.HallRequests)

	elevatorSystems[ourId] = localElevSystem
	localElevSystem = consensusTransition(ourId, elevatorSystems)
	return localElevSystem
}

// Should we check ALL requests for concesus, or just were we have "responsibility" for. E.g only our own cab requests or everyone elses aswell? I think both should work conceptually, but one of the solution might be better?
// Hall requests must be checked anyways always
// There should also be an easier way to find consensus than this. We are storing too much state and could maybe do/store this somewhere else?
// DO we need to add an extra number/state to the counter?? WHat happens if one elevator is stuck at unknown order, we have to sync somewhere else aswell. Can we sync on noOrder?
func consensusTransition(ourId string, elevatorSystems map[string]hra.ElevatorSystem) hra.ElevatorSystem {
	//Loop through all cab requests in our elevator, then compare all cab requests of all other elevators
cabNoconsensus: //Double check this placement
	for i, ourRequest := range elevatorSystems[ourId].ElevatorStates[ourId].CabRequests { //Double check what to do with unknown order here. If any order was unknown it shuld have pulled from network by now.
		if ourRequest != unknownOrder && ourRequest != servicedOrder {
			break
		}
		//Add all requests to array here. No need to think about our own requescabNoconsensusts as we know they will be the same
		for elevSystemId, elevSystem := range elevatorSystems {
			if ourRequest != elevSystem.ElevatorStates[ourId].CabRequests[i] {
				continue cabNoconsensus
			} else {
				//We have a consensus on unconfirmed request, transition to confirmed
				if ourRequest == unknownOrder {
					elevatorSystems[ourId].ElevatorStates[elevSystemId].CabRequests[i] = confirmedOrder
				} else if ourRequest == servicedOrder {
					elevatorSystems[ourId].ElevatorStates[elevSystemId].CabRequests[i] = noOrder
				}
			}
		}
	}

	for floor, row := range elevatorSystems[ourId].HallRequests { //Double check what to do with unknown order here. If any order was unknown it shuld have pulled from network by now.
	hallNoconsensus: //Double check placement here
		for btn, ourRequest := range row {
			if ourRequest != unknownOrder && ourRequest != servicedOrder {
				break
			}
			for _, elevSystem := range elevatorSystems {
				if ourRequest != elevSystem.HallRequests[floor][btn] {
					continue hallNoconsensus
				} else {
					//We have a consensus on unconfirmed request, transition to confirmed
					if ourRequest == unknownOrder {
						elevatorSystems[ourId].HallRequests[floor][btn] = confirmedOrder
					} else if ourRequest == servicedOrder {
						elevatorSystems[ourId].HallRequests[floor][btn] = noOrder
					}
				}
			}
		}

	}
	return elevatorSystems[ourId]
}

// These are very similar to the hraHallRequestTypeToBool and hraCabRequestTypeToBool. Consider merging them and passing modifier function
func transitionCabRequests(internalRequests []int, networkRequests []int) []int {
	newRequests := make([]int, len(internalRequests))
	for i, req := range internalRequests {
		newRequests[i] = transitionOrder(req, networkRequests[i])
	}
	return newRequests
}

func transitionHallRequests(internalRequests hra.HallRequestsType, networkRequests hra.HallRequestsType) hra.HallRequestsType {
	newRequests := make(hra.HallRequestsType, len(internalRequests))
	for i, row := range internalRequests {
		newRequests[i] = make([]int, len(row))
		for j, req := range row {
			newRequests[i][j] = transitionOrder(req, networkRequests[i][j]) //PROBLEM: We currently store requests as bools, but they must be ints. Maybe have them as ints until they go into HRA where confirmed orders are true, and everything else is false
		}
	}
	return newRequests
}

func transitionOrder(currentOrder int, newOrder int) int {
	if currentOrder == unknownOrder { //Catch up if we just joined
		return newOrder
	}
	if currentOrder == noOrder && newOrder == servicedOrder { //Prevent reset
		return currentOrder
	}
	if currentOrder == servicedOrder && newOrder == noOrder { //Reset
		return noOrder
	}
	if newOrder <= currentOrder { //Counter
		return currentOrder
	}
	return unknownOrder //Error state, to catch up to whatever network is doing.
}

func AddOrder(id string, elevatorSystem hra.ElevatorSystem, btn elevio.ButtonEvent) hra.ElevatorSystem {
	if btn.Button == elevio.BT_Cab {
		elevatorSystem.ElevatorStates[id].CabRequests[btn.Floor] = transitionOrder(elevatorSystem.ElevatorStates[id].CabRequests[btn.Floor], unconfirmedOrder)
	} else {
		elevatorSystem.HallRequests[btn.Floor][btn.Button] = transitionOrder(elevatorSystem.HallRequests[btn.Floor][btn.Button], unconfirmedOrder)
	}
	return elevatorSystem
}

func printElevatorSystem(elevatorSystem hra.ElevatorSystem) {
	fmt.Println("Elevator System State:")
	for id, state := range elevatorSystem.ElevatorStates {
		fmt.Printf("Elevator ID: %s\n", id)
		fmt.Printf("  Behaviour: %s, Floor: %d, Direction: %s\n", state.Behaviour, state.Floor, state.Direction)
		fmt.Println("  Cab Requests:")
		for i, req := range state.CabRequests {
			fmt.Printf("    Floor%d - %s\n", i, elevator.ReqToString(req))
		}
	}

	fmt.Println("Hall Requests:")
	// Reverse iteration of the HallRequests with a single loop
	for i := len(elevatorSystem.HallRequests) - 1; i >= 0; i-- {
		fmt.Printf("    Floor%d: ", i)
		for _, req := range elevatorSystem.HallRequests[i] {
			fmt.Printf("%s ", elevator.ReqToString(req))
		}
		fmt.Println()
	}
}
