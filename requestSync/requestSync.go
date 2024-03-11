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

type ElevatorState struct {
	Behaviour elevator.ElevatorBehaviour
	Floor     int
	Direction elevio.MotorDirection
}

type StateMsg struct {
	Id            string
	Counter       uint64 //Non-monotonic counter to only recieve newest data
	ElevatorState ElevatorState
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

type SyncOrder map[string]int //Map of what each elevator thinks the state of this order is (Could we reduce amount of state even more?

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

	timer := time.NewTimer(1 * time.Second)
	var msgCounter uint64 = 0

	elevatorSystems := make(map[string]ElevatorState)
	elevatorSystems[elevatorId] = ElevatorState{elevator.EB_Idle, -1, elevio.MD_Stop}
	tmp := <-elevatorSystemFromFSM

	var tmpElevator ElevatorState
	tmpElevator.Behaviour = tmp.Behaviour
	tmpElevator.Direction = tmp.Direction
	tmpElevator.Floor = tmp.Floor
	elevatorSystems[elevatorId] = tmpElevator

	syncOrderSystem := NewSyncOrderSystem(elevatorId)

	for {
		select {
		case btn := <-btnEvent: //Got order
			syncOrderSystem = AddOrder(elevatorId, syncOrderSystem, btn)
			msgCounter += 1 //To prevent forgetting counter, this should perhaps be in a seperate function
			//networkTransmitter <- StateMsg{elevatorId, msgCounter, elevatorSystems[elevatorId], SyncSystemToOrderSystem(elevatorId, syncOrderSystem)}

		case networkMsg := <-networkReciever: //TODO: Add elevatorSystem
			if networkMsg.Counter <= msgCounter && len(syncOrderSystem.CabRequests) != 1 && networkMsg.Id == elevatorId { //To only listen to our own message when we are alone
				msgCounter += 1
				break
			}
			elevatorSystems[networkMsg.Id] = networkMsg.ElevatorState
			msgCounter = networkMsg.Counter
			syncOrderSystem = Transition(elevatorId, networkMsg, syncOrderSystem)

			if elevatorSystems[elevatorId].Floor == -1 {
				//log.Println("Elevator floor is -1, will not send to hra")
				//log.Println("Behaviour: ", elevatorSystems[elevatorId].Behaviour)
				continue
			}

			hraOutput := hra.Decode(hra.AssignRequests(hra.Encode(SyncOrderSystemToElevatorSystem(elevatorSystems, elevatorId, syncOrderSystem))))[elevatorId]
			if len(hraOutput) > 0 {
				orderAssignment <- hraOutput
			} else {
				log.Println("Hra output empty, input to hra:", SyncOrderSystemToElevatorSystem(elevatorSystems, elevatorId, syncOrderSystem))
			}

		case peersUpdate := <-peersReciever:
			if len(peersUpdate.Peers) == 0 {
				//Set to unknown
				log.Println("Reset to unknown")
				syncOrderSystem = NewSyncOrderSystem(elevatorId)
			}

			updatedElevatorSystems := make(map[string]ElevatorState)
			for _, peers := range peersUpdate.Peers {
				updatedElevatorSystems[peers] = elevatorSystems[peers]
			}
			elevatorSystems = updatedElevatorSystems
		case elevator := <-elevatorSystemFromFSM:
			var tmpElevator ElevatorState
			tmpElevator.Behaviour = elevator.Behaviour
			tmpElevator.Direction = elevator.Direction
			tmpElevator.Floor = elevator.Floor
			elevatorSystems[elevatorId] = tmpElevator

		case orderToClear := <-orderCompleted:
			//Transmit to network that we want to clear
			//Bascially run a transition on our elevator system after having assigned the order as completed
			if orderToClear.Cab {
				syncOrderSystem.CabRequests[elevatorId][orderToClear.Floor][elevatorId] = noOrder
			}
			if orderToClear.HallUp {
				syncOrderSystem.HallRequests[orderToClear.Floor][0][elevatorId] = noOrder
			}
			if orderToClear.HallDown {
				syncOrderSystem.HallRequests[orderToClear.Floor][1][elevatorId] = noOrder
			}

		case <-timer.C: //Timer reset, send new state update
			msgCounter += 1
			networkTransmitter <- StateMsg{elevatorId, msgCounter, elevatorSystems[elevatorId], SyncSystemToOrderSystem(elevatorId, syncOrderSystem)}
			timer.Reset(time.Millisecond * 1000)
		}
	}
}

func AddOrder(ourId string, syncOrderSystem SyncOrderSystem, btn elevio.ButtonEvent) SyncOrderSystem {
	if btn.Button == elevio.BT_Cab {
		syncOrderSystem.CabRequests[ourId][btn.Floor][ourId] = TransitionOrder(syncOrderSystem.CabRequests[ourId][btn.Floor][ourId], unconfirmedOrder)
	} else {
		syncOrderSystem.HallRequests[btn.Floor][btn.Button][ourId] = TransitionOrder(syncOrderSystem.HallRequests[btn.Floor][btn.Button][ourId], unconfirmedOrder)
	}
	return syncOrderSystem
}

func Transition(ourId string, networkMsg StateMsg, syncOrderSystem SyncOrderSystem) SyncOrderSystem {
	syncOrderSystem = AddElevatorToSyncOrderSystem(ourId, networkMsg, syncOrderSystem)
	orderSystem := SyncSystemToOrderSystem(ourId, syncOrderSystem)
	orderSystem.HallRequests = TransitionHallRequests(orderSystem.HallRequests, networkMsg.OrderSystem.HallRequests)

	_, ok := syncOrderSystem.CabRequests[networkMsg.Id]
	if ok && len(orderSystem.CabRequests[networkMsg.Id]) > 0 && len(orderSystem.CabRequests[ourId]) > 0 {
		//log.Println("Transitioned cabs from", networkMsg.Id, "in", ourId)
		orderSystem.CabRequests[ourId] = TransitionCabRequests(orderSystem.CabRequests[ourId], orderSystem.CabRequests[networkMsg.Id])
	} else {
		log.Println("Could not transition cabs. Elevator", networkMsg.Id, "does not have our (", ourId, ") requests. Have they received our state?")
		//log.Println(networkMsg.OrderSystem.CabRequests)
	}
	syncOrderSystem = systemToSyncOrderSystem(ourId, syncOrderSystem, orderSystem)

	return ConsensusBarrierTransition(ourId, syncOrderSystem)
}

// TODO: Add unit tests for this functon.
func AddElevatorToSyncOrderSystem(ourId string, networkMsg StateMsg, syncOrderSystem SyncOrderSystem) SyncOrderSystem {
	//Update our records of the view networkElevator of our cabs
	for floor, networkRequest := range networkMsg.OrderSystem.CabRequests[ourId] {
		syncOrderSystem.CabRequests[ourId][floor][networkMsg.Id] = networkRequest
	}
	//Update our records of the view networkElevator our
	for floor, row := range networkMsg.OrderSystem.HallRequests {
		for btn, networkRequest := range row {
			syncOrderSystem.HallRequests[floor][btn][networkMsg.Id] = networkRequest
		}
	}

	_, ok := syncOrderSystem.CabRequests[networkMsg.Id]
	if !ok {
		syncOrderSystem.CabRequests[networkMsg.Id] = make([]SyncOrder, len(syncOrderSystem.CabRequests[ourId]))
	}
	//Add/Update the cab requests of the other elevator into our own representation of them.
	//Because we only add to our own representation here, we could have just used int for this. We could do this because we never run a consensus transition on anyone elses cab requests.
	for floor, req := range networkMsg.OrderSystem.CabRequests[networkMsg.Id] {
		syncOrderSystem.CabRequests[networkMsg.Id][floor] = make(SyncOrder)
		syncOrderSystem.CabRequests[networkMsg.Id][floor][networkMsg.Id] = req
	}

	return syncOrderSystem
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

func ConsensusBarrierTransition(ourId string, OrderSystem SyncOrderSystem) SyncOrderSystem {
	//Transition all cabs
	{
		floor, newState := ConsensusTransitionSingleCab(ourId, OrderSystem.CabRequests)
		for floor != -1 && newState != -1 { //Can this create nasty edge cases? Maybe have a validation test earlier to check for unknown orders
			OrderSystem.CabRequests[ourId][floor][ourId] = newState
			floor, newState = ConsensusTransitionSingleCab(ourId, OrderSystem.CabRequests)
		}
	}

	//Transition all halls
	{
		floor, btn, newState := ConsensusTransitionSingleHall(ourId, OrderSystem.HallRequests)
		for floor != -1 && btn != -1 {
			OrderSystem.HallRequests[floor][btn][ourId] = newState
			floor, btn, newState = ConsensusTransitionSingleHall(ourId, OrderSystem.HallRequests)
		}
	}
	return OrderSystem
}

// Cab and hall are very similar, we should refactor more
func ConsensusTransitionSingleCab(ourId string, cabRequests map[string][]SyncOrder) (int, int) {
	for reqFloor, req := range cabRequests[ourId] { //We only check our own cabs for consensus
		if AllValuesEqual(req) { //Consensus
			ourRequest := req[ourId]
			if ourRequest == servicedOrder {
				return reqFloor, noOrder
			} else if ourRequest == unconfirmedOrder {
				return reqFloor, confirmedOrder
			}
		}
	}
	return -1, -1
}

func ConsensusTransitionSingleHall(ourId string, hallRequests [][]SyncOrder) (int, int, int) { // (int, int ,int) is not clear, should have order type instead
	for reqFloor, row := range hallRequests {
		for reqBtn, req := range row {
			if AllValuesEqual(req) {
				ourRequest := req[ourId]
				if ourRequest == servicedOrder {
					return reqFloor, reqBtn, noOrder
				} else if ourRequest == unconfirmedOrder {
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
	initMap = make(map[string]int)
	initMap[initialKey] = unknownOrder // Init with unknown to just join NW
	hallRequests := [][]SyncOrder{{initMap, initMap}, {initMap, initMap}, {initMap, initMap}, {initMap, initMap}}
	for i := 0; i < 4; i++ {
		for j := 0; j < 2; j++ {
			initMap = make(map[string]int)
			initMap[initialKey] = unknownOrder // Init with unknown to just join NW
			hallRequests[i][j] = map[string]int{initialKey: unknownOrder}
		}
	}

	// Initialize CabRequests with 4 SyncOrder elements for each key.
	cabRequests := map[string][]SyncOrder{initialKey: {initMap, initMap, initMap, initMap}}
	for i := 0; i < 4; i++ {
		initMap = make(map[string]int)
		initMap[initialKey] = unknownOrder // Init with unknown to just join NW
		cabRequests[initialKey][i] = initMap
	}
	return SyncOrderSystem{
		HallRequests: hallRequests,
		CabRequests:  cabRequests,
	}
}

func newOrderSystem(id string) OrderSystem {
	cabRequests := make(map[string][]int)
	cabRequests[id] = make([]int, 4)
	for i := 0; i < 4; i++ {
		cabRequests[id][i] = unknownOrder
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
			syncOrderSystem.HallRequests[i][j][ourId] = req
		}
	}
	for i, req := range orderSystem.CabRequests[ourId] {
		syncOrderSystem.CabRequests[ourId][i][ourId] = req
	}
	return syncOrderSystem
}

func SyncSystemToOrderSystem(ourId string, orderSystem SyncOrderSystem) OrderSystem {
	var newOrderSystem OrderSystem = newOrderSystem(ourId)

	for i, floor := range orderSystem.HallRequests {
		for j, req := range floor {
			newOrderSystem.HallRequests[i][j] = req[ourId]
		}
	}

	for id, cabs := range orderSystem.CabRequests {
		for i, req := range cabs {
			newOrderSystem.CabRequests[ourId][i] = req[id]
		}
	}

	return newOrderSystem
}

func GenerateLocalElev(elevatorSystem ElevatorState, id string, OrderSystem SyncOrderSystem) hra.LocalElevatorState {
	localElevState := hra.LocalElevatorState{
		Behaviour:   elevatorSystem.Behaviour,
		Floor:       elevatorSystem.Floor,
		Direction:   elevatorSystem.Direction,
		CabRequests: make([]int, len(OrderSystem.CabRequests[id])),
	}

	for i, req := range OrderSystem.CabRequests[id] {
		localElevState.CabRequests[i] = req[id]
	}

	return localElevState
}

func SyncOrderSystemToElevatorSystem(elevatorSystems map[string]ElevatorState, ourId string, OrderSystem SyncOrderSystem) hra.ElevatorSystem {
	hraElevSys := hra.ElevatorSystem{
		HallRequests:   [][]int{{unknownOrder, unknownOrder}, {unknownOrder, unknownOrder}, {unknownOrder, unknownOrder}, {unknownOrder, unknownOrder}},
		ElevatorStates: map[string]hra.LocalElevatorState{},
	}

	//Fill halls
	for i, floor := range OrderSystem.HallRequests {
		for j, req := range floor {
			hraElevSys.HallRequests[i][j] = req[ourId]
		}
	}
	//Loop through all IDs and add elevatorsystem pr id
	for id, _ := range OrderSystem.CabRequests {
		hraElevSys.ElevatorStates[id] = GenerateLocalElev(elevatorSystems[id], id, OrderSystem)
	}

	return hraElevSys
}

// These are very similar to the hraHallRequestTypeToBool and hraCabRequestTypeToBool. Consider merging them and passing modifier function
func TransitionCabRequests(internalRequests []int, networkRequests []int) []int {
	fmt.Println(internalRequests)
	fmt.Println(networkRequests)
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
