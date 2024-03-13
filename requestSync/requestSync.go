package requestSync

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/hra"
	"elevatorAlgorithm/requests"
	"elevatorDriver/elevio"
	"log"
	"networkDriver/bcast"
	"networkDriver/peers"
	"time"
)

type Elevator struct {
	Behaviour elevator.ElevatorBehaviour
	Floor     int
	Direction elevio.MotorDirection
}

type StateMsg struct {
	Id            string
	Counter       uint64 //Non-monotonic counter to only recieve newest data
	ElevatorState Elevator
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

type SyncOrder map[string]int //Map of what each elevator thinks the state of this order is (Could we reduce amount of state even more?

type SyncOrderSystem struct {
	HallRequests [][]SyncOrder
	CabRequests  map[string][]SyncOrder
}
type OrderSystem struct {
	HallRequests [][]int
	CabRequests  map[string][]int
}

func Sync(elevatorSystemFromFSM chan elevator.Elevator, localId string, orderAssignment chan [][]bool, orderCompleted chan requests.ClearFloorOrders, peersReciever chan peers.PeerUpdate) {
	btnEvent := make(chan elevio.ButtonEvent)
	networkReciever := make(chan StateMsg)
	networkTransmitter := make(chan StateMsg)

	go bcast.Receiver(bcastPort, networkReciever)
	go bcast.Transmitter(bcastPort, networkTransmitter)

	go elevio.PollButtons(btnEvent)

	timer := time.NewTimer(100 * time.Millisecond)
	var msgCounter uint64 = 0

	elevatorSystems := make(map[string]Elevator)
	elevatorSystems[localId] = Elevator{elevator.EB_Idle, -1, elevio.MD_Stop}
	tmp := <-elevatorSystemFromFSM

	var tmpElevator Elevator
	tmpElevator.Behaviour = tmp.Behaviour
	tmpElevator.Direction = tmp.Direction
	tmpElevator.Floor = tmp.Floor
	elevatorSystems[localId] = tmpElevator

	syncOrderSystem := NewSyncOrderSystem(localId)
	log.Println(syncOrderSystem)
	var activePeers []string

	for {
		select {
		case btn := <-btnEvent:
			syncOrderSystem = AddOrder(localId, syncOrderSystem, btn)
			msgCounter += 1 //To prevent forgetting counter, this should perhaps be in a seperate function
			networkTransmitter <- StateMsg{localId, msgCounter, elevatorSystems[localId], SyncOrderSystemToOrderSystem(localId, syncOrderSystem)}

		case networkMsg := <-networkReciever:
			if networkMsg.Counter <= msgCounter && len(activePeers) != 1 && networkMsg.Id == localId { //To only listen to our own message when we are alone
				msgCounter += 1
				break
			}
			elevatorSystems[networkMsg.Id] = networkMsg.ElevatorState
			msgCounter = networkMsg.Counter
			syncOrderSystem = Transition(localId, networkMsg, syncOrderSystem, activePeers)

			if elevatorSystems[localId].Floor == -1 {
				log.Println("Elevator floor is -1, will not send to hra")
				break
			}
			elevatorSystem := SyncOrderSystemToElevatorSystem(elevatorSystems, localId, syncOrderSystem, activePeers)
			updateHallLights(elevatorSystem.HallRequests)
			updateCabLights(elevatorSystem.ElevatorStates[localId].CabRequests)
			hraOutput := hra.Decode(hra.AssignRequests(hra.Encode(elevatorSystem)))[localId]
			if len(hraOutput) > 0 {
				select {
				case orderAssignment <- hraOutput:
				default:
				}
			} else {
				log.Println("Hra output empty, input to hra (There could be invalid peers which are not sent here):", SyncOrderSystemToElevatorSystem(elevatorSystems, localId, syncOrderSystem, activePeers))
			}

		case peersUpdate := <-peersReciever:
			activePeers = peersUpdate.Peers
			log.Println("Peers:", peersUpdate.Peers, "Elevsys:", len(elevatorSystems), "syncOrderSystem cabs num:", len(syncOrderSystem.CabRequests), "syncOrderSys specific orders: ", len(syncOrderSystem.CabRequests[localId][0]))
		case elevator := <-elevatorSystemFromFSM:
			var tmpElevator Elevator
			tmpElevator.Behaviour = elevator.Behaviour
			tmpElevator.Direction = elevator.Direction
			tmpElevator.Floor = elevator.Floor
			elevatorSystems[localId] = tmpElevator

		case orderToClear := <-orderCompleted:
			syncOrderSystem = RemoveOrder(localId, orderToClear, syncOrderSystem)

		case <-timer.C: //Timer reset, send new state update
			msgCounter += 1
			networkTransmitter <- StateMsg{localId, msgCounter, elevatorSystems[localId], SyncOrderSystemToOrderSystem(localId, syncOrderSystem)}
			timer.Reset(time.Millisecond * 10)
		}
	}
}

func AddOrder(localId string, syncOrderSystem SyncOrderSystem, btn elevio.ButtonEvent) SyncOrderSystem {
	if btn.Button == elevio.BT_Cab {
		syncOrderSystem.CabRequests[localId][btn.Floor][localId] = TransitionOrder(syncOrderSystem.CabRequests[localId][btn.Floor][localId], unconfirmedOrder)
	} else {
		syncOrderSystem.HallRequests[btn.Floor][btn.Button][localId] = TransitionOrder(syncOrderSystem.HallRequests[btn.Floor][btn.Button][localId], unconfirmedOrder)
	}
	return syncOrderSystem
}
func RemoveOrder(localId string, orderToClear requests.ClearFloorOrders, syncOrderSystem SyncOrderSystem) SyncOrderSystem {
	if orderToClear.Cab {
		syncOrderSystem.CabRequests[localId][orderToClear.Floor][localId] = TransitionOrder(syncOrderSystem.CabRequests[localId][orderToClear.Floor][localId], servicedOrder)
	}
	if orderToClear.HallUp {
		syncOrderSystem.HallRequests[orderToClear.Floor][0][localId] = TransitionOrder(syncOrderSystem.HallRequests[orderToClear.Floor][0][localId], servicedOrder)
	}
	if orderToClear.HallDown {
		syncOrderSystem.HallRequests[orderToClear.Floor][1][localId] = TransitionOrder(syncOrderSystem.HallRequests[orderToClear.Floor][1][localId], servicedOrder)
	}
	return syncOrderSystem
}

func Transition(localId string, networkMsg StateMsg, updatedSyncOrderSystem SyncOrderSystem, peers []string) SyncOrderSystem {
	updatedSyncOrderSystem = AddElevatorToSyncOrderSystem(localId, networkMsg, updatedSyncOrderSystem)

	orderSystem := SyncOrderSystemToOrderSystem(localId, updatedSyncOrderSystem)
	orderSystem.HallRequests = TransitionHallRequests(orderSystem.HallRequests, networkMsg.OrderSystem.HallRequests)

	_, ok := updatedSyncOrderSystem.CabRequests[networkMsg.Id]
	if ok && len(orderSystem.CabRequests[networkMsg.Id]) > 0 {
		orderSystem.CabRequests[localId] = TransitionCabRequests(orderSystem.CabRequests[localId], orderSystem.CabRequests[networkMsg.Id])
	} else {
		log.Println("Could not transition cabs. We did not add elevator", networkMsg.Id, "to syncOrderSystem")
	}
	updatedSyncOrderSystem = UpdateSyncOrderSystem(localId, updatedSyncOrderSystem, orderSystem)

	return ConsensusBarrierTransition(localId, updatedSyncOrderSystem, peers)
}

// Should not need peer list here, if we got networkMsg, the elevator is in peer list
func AddElevatorToSyncOrderSystem(localId string, networkMsg StateMsg, syncOrderSystem SyncOrderSystem) SyncOrderSystem {
	//Update our records of the view networkElevator has of our cabs
	for floor, networkRequest := range networkMsg.OrderSystem.CabRequests[localId] {
		syncOrderSystem.CabRequests[localId][floor][networkMsg.Id] = networkRequest
	}
	//Update our records of the view networkElevator has of halls
	for floor, requests := range networkMsg.OrderSystem.HallRequests {
		for btn, networkRequest := range requests {
			syncOrderSystem.HallRequests[floor][btn][networkMsg.Id] = networkRequest
		}
	}

	_, ok := syncOrderSystem.CabRequests[networkMsg.Id]
	if !ok {
		syncOrderSystem.CabRequests[networkMsg.Id] = make([]SyncOrder, len(syncOrderSystem.CabRequests[localId]))
	}
	//Add/Update the cab requests of the other elevator into our own representation of them.
	//Because we only add to our own representation here, we could have just used int for this. We could do this because we never run a consensus transition on anyone elses cab requests.
	//This also means accessing syncOrderSystem.CabRequests[networkMsg.Id][floor][NETWORKID] is always wrong
	for floor, req := range networkMsg.OrderSystem.CabRequests[networkMsg.Id] {
		syncOrderSystem.CabRequests[networkMsg.Id][floor] = make(SyncOrder)
		syncOrderSystem.CabRequests[networkMsg.Id][floor][localId] = req //This only adds
	}
	return syncOrderSystem
}

func TransitionOrder(currentOrder int, updatedOrder int) int {
	if currentOrder == unknownOrder { //Catch up if we just joined
		return updatedOrder
	}
	if currentOrder == noOrder && updatedOrder == servicedOrder { //Prevent reset
		return currentOrder
	}
	if currentOrder == servicedOrder && updatedOrder == noOrder { //Reset
		return noOrder
	}
	if updatedOrder <= currentOrder { //Counter
		return currentOrder
	} else {
		return updatedOrder
	}
}

func ConsensusBarrierTransition(localId string, OrderSystem SyncOrderSystem, peers []string) SyncOrderSystem {
	floor, newState := ConsensusTransitionSingleCab(localId, OrderSystem.CabRequests, peers)
	if floor != -1 && newState != -1 {
		OrderSystem.CabRequests[localId][floor][localId] = newState
	}

	floor, btn, newState := ConsensusTransitionSingleHall(localId, OrderSystem.HallRequests, peers)
	if floor != -1 && btn != -1 {
		OrderSystem.HallRequests[floor][btn][localId] = newState
	}
	return OrderSystem
}

// Cab and hall are very similar, we should refactor more
func ConsensusTransitionSingleCab(localId string, cabRequests map[string][]SyncOrder, peers []string) (int, int) {
	for reqFloor, req := range cabRequests[localId] { //We only check our own cabs for consensus
		if AllPeersHaveSameValue(req, peers) { //Consensus
			ourRequest := req[localId]
			if ourRequest == servicedOrder {
				return reqFloor, noOrder
			} else if ourRequest == unconfirmedOrder {
				return reqFloor, confirmedOrder
			}
		}
	}
	return -1, -1
}

func ConsensusTransitionSingleHall(localId string, hallRequests [][]SyncOrder, peers []string) (int, int, int) { // (int, int ,int) is not clear, should have order type instead
	for reqFloor, row := range hallRequests {
		for reqBtn, req := range row {
			if AllPeersHaveSameValue(req, peers) {
				ourRequest := req[localId]
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

func AllPeersHaveSameValue(order SyncOrder, peers []string) bool {
	var firstValue int
	isFirst := true

	for _, peerId := range peers {
		value, ok := order[peerId]
		if !ok {
			return false //If we dont have the order, we dont have consensus
		}
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

func NewSyncOrderSystem(id string) SyncOrderSystem {
	hallRequests := make([][]SyncOrder, elevator.N_FLOORS)
	for i := 0; i < elevator.N_FLOORS; i++ {
		hallRequests[i] = make([]SyncOrder, elevator.N_HALL_BUTTONS)
		for j := 0; j < elevator.N_HALL_BUTTONS; j++ {
			initMap := make(SyncOrder)
			initMap[id] = unknownOrder // Init with unknown to just join network
			hallRequests[i][j] = map[string]int{id: unknownOrder}
		}
	}

	cabRequests := make(map[string][]SyncOrder)
	cabRequests[id] = make([]SyncOrder, elevator.N_FLOORS)
	for i := 0; i < elevator.N_FLOORS; i++ { //Fix
		cabRequests[id][i] = SyncOrder{id: unknownOrder}
	}
	return SyncOrderSystem{
		HallRequests: hallRequests,
		CabRequests:  cabRequests,
	}
}

func newOrderSystem(id string) OrderSystem {
	hallRequests := make([][]int, elevator.N_FLOORS)
	for i := 0; i < elevator.N_FLOORS; i++ {
		hallRequests[i] = make([]int, elevator.N_HALL_BUTTONS)
	}

	cabRequests := make(map[string][]int)
	cabRequests[id] = make([]int, elevator.N_FLOORS)
	for i := 0; i < elevator.N_FLOORS; i++ {
		cabRequests[id][i] = unknownOrder
	}

	return OrderSystem{
		HallRequests: hallRequests,
		CabRequests:  cabRequests,
	}
}

func UpdateSyncOrderSystem(localId string, syncOrderSystem SyncOrderSystem, orderSystem OrderSystem) SyncOrderSystem {
	for i, floor := range orderSystem.HallRequests {
		for j, req := range floor {
			syncOrderSystem.HallRequests[i][j][localId] = req
		}
	}
	for id, cabs := range syncOrderSystem.CabRequests {
		for floor, req := range cabs {
			syncOrderSystem.CabRequests[id][floor][localId] = req[localId]
		}
	}
	return syncOrderSystem
}

func SyncOrderSystemToOrderSystem(localId string, syncOrderSystem SyncOrderSystem) OrderSystem {
	var newOrderSystem OrderSystem = newOrderSystem(localId)

	for i, floor := range syncOrderSystem.HallRequests {
		for j, req := range floor {
			newOrderSystem.HallRequests[i][j] = req[localId]
		}
	}
	for id, cabs := range syncOrderSystem.CabRequests {
		newOrderSystem.CabRequests[id] = make([]int, elevator.N_FLOORS)
		for i, req := range cabs {
			newOrderSystem.CabRequests[id][i] = req[localId] //This is dangrous but needed. See comments in AddElevatorToSyncOrderSystem. This could be the only place where we access SyncOrderSystem, but only care about ourself (e.g we could use orderSystem all other places)
		}
	}

	return newOrderSystem
}

func GenerateLocalElev(elevatorSystem Elevator, id string, OrderSystem SyncOrderSystem) hra.LocalElevatorState {
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

func SyncOrderSystemToElevatorSystem(elevatorSystems map[string]Elevator, localId string, OrderSystem SyncOrderSystem, peers []string) hra.ElevatorSystem {
	hraElevSys := hra.ElevatorSystem{
		HallRequests:   [][]int{{unknownOrder, unknownOrder}, {unknownOrder, unknownOrder}, {unknownOrder, unknownOrder}, {unknownOrder, unknownOrder}},
		ElevatorStates: map[string]hra.LocalElevatorState{},
	}

	//Fill halls. Don't need peers, because we base it on our own id
	for i, floor := range OrderSystem.HallRequests {
		for j, req := range floor {
			hraElevSys.HallRequests[i][j] = req[localId]
		}
	}

	//We only want to add alive peers to HRA input
	for _, id := range peers {
		hraElevSys.ElevatorStates[id] = GenerateLocalElev(elevatorSystems[id], id, OrderSystem)
	}

	return hraElevSys
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

// Move these?
func updateHallLights(hall_orders [][]int) {
	for floor, floorRow := range hall_orders {
		for btn, order := range floorRow {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, (order == confirmedOrder))
		}
	}
}

func updateCabLights(cab_orders []int) {
	for floor, order := range cab_orders {
		elevio.SetButtonLamp(elevio.BT_Cab, floor, (order == confirmedOrder))
	}
}
