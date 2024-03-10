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

type elevatorState struct {
	Behaviour elevator.ElevatorBehaviour
	Floor     int
	Direction elevio.MotorDirection
}

type StateMsg struct {
	Id            string
	Counter       uint64 //Non-monotonic counter to only recieve newest data
	ElevatorState elevatorState
	OrderSystem   OrderSystem
}

// Must be imported?
const (
	unknownOrder = iota
	noOrder
	unconfirmedOrder
	confirmedOrder
	servicedOrder
)
const bcastPort int = 25565
const peersPort int = 25566

type SyncOrder struct { // Bad name, order status is better
	OrderState map[string]int //Map of what each elevator thinks the state of this order is (Could we reduce amount of state even more? In concensusTransition we only care about if a state is equal to our own)
}

type SyncOrderSystem struct {
	HallRequests [][]SyncOrder
	CabRequests  map[string][]SyncOrder
}
type OrderSystem struct {
	HallRequests [][]int
	CabRequests  map[string][]int
}

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
	var msgCounter uint64 = 0
	var latestPeerList []string

	elevatorSystem := elevatorState{elevator.EB_Idle, -1, elevio.MD_Stop}
	syncOrderSystem := NewSyncOrderSystem(elevatorId)

	for {
		select {
		case btn := <-btnEvent: //Got order
			syncOrderSystem = AddOrder(elevatorId, syncOrderSystem, btn)
			msgCounter += 1 //To prevent forgetting counter, this should perhaps be in a seperate function
			networkTransmitter <- StateMsg{elevatorId, msgCounter, elevatorSystem, syncSystemToOrderSystem(elevatorId, syncOrderSystem)}

		case networkMsg := <-networkReciever: //Got message
			if networkMsg.Counter <= msgCounter {
				msgCounter += 1
				break
			}
			msgCounter = networkMsg.Counter
			syncOrderSystem = Transition(elevatorId, networkMsg, syncOrderSystem)
			orderAssignment <- hra.Decode(hra.AssignRequests(hra.Encode(SyncOrderSystemToElevatorSystem(elevatorSystem, elevatorId, syncOrderSystem))))[elevatorId]
		case peersUpdate := <-peersReciever:
			latestPeerList = peersUpdate.Peers //Here we should also update the elevatorSystem map. Important to take the (hall)orders of lost peers before removing it
			_ = latestPeerList
		case elevator := <-elevatorSystemFromFSM:
			elevatorSystem.Behaviour = elevator.Behaviour
			elevatorSystem.Direction = elevator.Direction
			elevatorSystem.Floor = elevator.Floor

		case orderToClear := <-orderCompleted:
			//Transmit to network that we want to clear
			//Bascially run a transition on our elevator system after having assigned the order as completed
			_ = orderToClear
		case <-timer.C: //Timer reset, send new state update
			msgCounter += 1
			networkTransmitter <- StateMsg{elevatorId, msgCounter, elevatorSystem, syncSystemToOrderSystem(elevatorId, syncOrderSystem)}
			timer.Reset(time.Millisecond * 1000)
		}
	}
}

func AddOrder(ourId string, syncOrderSystem SyncOrderSystem, btn elevio.ButtonEvent) SyncOrderSystem {
	if btn.Button == elevio.BT_Cab {
		syncOrderSystem.CabRequests[ourId][btn.Floor].OrderState[ourId] = TransitionOrder(syncOrderSystem.CabRequests[ourId][btn.Floor].OrderState[ourId], unconfirmedOrder)
	} else {
		syncOrderSystem.HallRequests[btn.Floor][btn.Button].OrderState[ourId] = TransitionOrder(syncOrderSystem.HallRequests[btn.Floor][btn.Button].OrderState[ourId], unconfirmedOrder)
	}
	fmt.Println("Button: ", btn)
	fmt.Println("Cabs: ", syncOrderSystem.CabRequests)
	fmt.Println("Halls: ", syncOrderSystem.HallRequests)

	return syncOrderSystem
}

func Transition(ourId string, networkMsg StateMsg, syncOrderSystem SyncOrderSystem) SyncOrderSystem {
	orderSystem := syncSystemToOrderSystem(ourId, syncOrderSystem)
	orderSystem.HallRequests = TransitionHallRequests(orderSystem.HallRequests, networkMsg.OrderSystem.HallRequests)
	orderSystem.CabRequests[ourId] = TransitionCabRequests(orderSystem.CabRequests[ourId], networkMsg.OrderSystem.CabRequests[ourId])
	syncOrderSystem = systemToSyncOrderSystem(ourId, syncOrderSystem, orderSystem)

	return ConsensusBarrierTransition(ourId, syncOrderSystem)
}

func TransitionOrder(currentOrder int, newOrder int) int {
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
	} else {
		return newOrder
	}
}

// Cab and hall are very similar, we should refactor more
func ConsensusTransitionSingleCab(ourId string, cabRequests map[string][]SyncOrder) (int, int) {
	for reqFloor, req := range cabRequests[ourId] { //We only check our own cabs for consensus
		if AllValuesEqual(req.OrderState) { //Consensus
			ourRequest := req.OrderState[ourId]
			if ourRequest == servicedOrder {
				return reqFloor, noOrder
			} else if ourRequest == unknownOrder {
				return reqFloor, confirmedOrder
			}
		}
	}
	return -1, -1
}

func ConsensusTransitionSingleHall(ourId string, hallRequests [][]SyncOrder) (int, int, int) { // (int, int ,int) is not clear, should have order type instead
	for reqFloor, row := range hallRequests {
		for reqBtn, req := range row {
			if AllValuesEqual(req.OrderState) {
				ourRequest := req.OrderState[ourId]
				if ourRequest == servicedOrder {
					return reqFloor, reqBtn, noOrder
				} else if ourRequest == unknownOrder {
					return reqFloor, reqBtn, confirmedOrder
				}
			}
		}
	}
	return -1, -1, -1
}

func NewSyncOrderSystem(initialKey string) SyncOrderSystem {
	// Initialize HallRequests with fixed sizes.
	initMap := SyncOrder{}
	initMap.OrderState = make(map[string]int)
	initMap.OrderState[initialKey] = unknownOrder // Init with unknown to just join NW
	hallRequests := [][]SyncOrder{{initMap, initMap}, {initMap, initMap}, {initMap, initMap}, {initMap, initMap}}
	for i := 0; i < 4; i++ {
		for j := 0; j < 2; j++ {
			initMap.OrderState = make(map[string]int)
			initMap.OrderState[initialKey] = unknownOrder // Init with unknown to just join NW
			hallRequests[i][j] = SyncOrder{OrderState: map[string]int{initialKey: unknownOrder}}
		}
	}

	// Initialize CabRequests with 4 SyncOrder elements for each key.
	cabRequests := map[string][]SyncOrder{initialKey: {initMap, initMap, initMap, initMap}}
	for i := 0; i < 4; i++ {
		initMap.OrderState = make(map[string]int)
		initMap.OrderState[initialKey] = unknownOrder // Init with unknown to just join NW
		cabRequests[initialKey][i] = initMap
	}
	log.Println("Made cabRequests:", cabRequests)
	return SyncOrderSystem{
		HallRequests: hallRequests,
		CabRequests:  cabRequests,
	}
}

func newOrderSystem(id string) OrderSystem {
	cabRequests := make(map[string][]int)
	for i := 0; i < 4; i++ {
		cabRequests[id] = append(cabRequests[id], unknownOrder)
	}

	return OrderSystem{
		HallRequests: [][]int{{unknownOrder, unknownOrder}, {unknownOrder, unknownOrder}, {unknownOrder, unknownOrder}, {unknownOrder, unknownOrder}},
		CabRequests:  cabRequests,
	}
}

func AllValuesEqual(m map[string]int) bool {
	var firstValue int
	isFirst := true

	for _, value := range m {
		if isFirst {
			firstValue = value
			isFirst = false
		} else {
			if value != firstValue {
				return false
			}
		}
	}
	return true
}

func systemToSyncOrderSystem(ourId string, syncOrderSystem SyncOrderSystem, orderSystem OrderSystem) SyncOrderSystem {
	for i, floor := range orderSystem.HallRequests {
		for j, req := range floor {
			syncOrderSystem.HallRequests[i][j].OrderState[ourId] = req
		}
	}
	for i, req := range orderSystem.CabRequests[ourId] {
		syncOrderSystem.CabRequests[ourId][i].OrderState[ourId] = req
	}

	return syncOrderSystem
}

func syncSystemToOrderSystem(ourId string, orderSystem SyncOrderSystem) OrderSystem {
	var orderSys OrderSystem = newOrderSystem(ourId)

	for i, floor := range orderSystem.HallRequests {
		for j, req := range floor {
			orderSys.HallRequests[i][j] = req.OrderState[ourId]
		}
	}
	for i, req := range orderSystem.CabRequests[ourId] {
		orderSys.CabRequests[ourId][i] = req.OrderState[ourId]
	}
	return orderSys
}

func SyncOrderSystemToElevatorSystem(elevatorSystem elevatorState, ourId string, OrderSystem SyncOrderSystem) hra.ElevatorSystem {
	var hraElevSys hra.ElevatorSystem
	tmpLocalElevatorState := hra.LocalElevatorState{}
	tmpLocalElevatorState.Behaviour = elevatorSystem.Behaviour
	tmpLocalElevatorState.Direction = elevatorSystem.Direction
	tmpLocalElevatorState.Floor = elevatorSystem.Floor
	hraElevSys.ElevatorStates[ourId] = tmpLocalElevatorState

	for i, floor := range OrderSystem.HallRequests {
		for j, req := range floor {
			hraElevSys.HallRequests[i][j] = req.OrderState[ourId]
		}
	}
	for i, req := range OrderSystem.CabRequests[ourId] {
		hraElevSys.ElevatorStates[ourId].CabRequests[i] = req.OrderState[ourId]
	}
	return hraElevSys
}

func ConsensusBarrierTransition(ourId string, OrderSystem SyncOrderSystem) SyncOrderSystem {
	//Transition all cabs
	{
		floor, newState := ConsensusTransitionSingleCab(ourId, OrderSystem.CabRequests)
		for floor != -1 && newState != -1 { //Can this create nasty edge cases? Maybe have a validation test earlier to check for unknown orders
			OrderSystem.CabRequests[ourId][floor].OrderState[ourId] = newState
			floor, newState = ConsensusTransitionSingleCab(ourId, OrderSystem.CabRequests)
		}
	}

	//Transition all halls
	{
		floor, btn, newState := ConsensusTransitionSingleHall(ourId, OrderSystem.HallRequests)
		for floor != -1 && btn != -1 {
			OrderSystem.HallRequests[floor][btn].OrderState[ourId] = newState
			floor, btn, newState = ConsensusTransitionSingleHall(ourId, OrderSystem.HallRequests)
		}
	}
	return OrderSystem
}

// These are very similar to the hraHallRequestTypeToBool and hraCabRequestTypeToBool. Consider merging them and passing modifier function
func TransitionCabRequests(internalRequests []int, networkRequests []int) []int {
	for i, req := range internalRequests {
		internalRequests[i] = TransitionOrder(req, networkRequests[i])
	}
	return internalRequests
}

func TransitionHallRequests(internalRequests [][]int, networkRequests [][]int) [][]int {
	for i, row := range internalRequests {
		for j, req := range row {
			internalRequests[i][j] = TransitionOrder(req, networkRequests[i][j])
		}
	}
	return internalRequests
}

func ReqToString(req int) string {
	switch req {
	case unknownOrder:
		return "unknown"
	case noOrder:
		return "no request"
	case unconfirmedOrder:
		return "unconfirmed"
	case confirmedOrder:
		return "confirmed"
	case servicedOrder:
		return "serviced order"
	default:
		return "Invalid"
	}
}
