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

// Shit name: elevatorId is ambiguous, who's ID? Ours? Network? Local?
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

	var activePeers []string

	for {
		select {
		case btn := <-btnEvent: //Got order
			syncOrderSystem = AddOrder(elevatorId, syncOrderSystem, btn)
			msgCounter += 1 //To prevent forgetting counter, this should perhaps be in a seperate function
			networkTransmitter <- StateMsg{elevatorId, msgCounter, elevatorSystems[elevatorId], SyncOrderSystemToOrderSystem(elevatorId, syncOrderSystem)}

		case networkMsg := <-networkReciever: //TODO: Add elevatorSystem
			if networkMsg.Counter <= msgCounter && len(syncOrderSystem.CabRequests) != 1 && networkMsg.Id == elevatorId { //To only listen to our own message when we are alone
				msgCounter += 1
				continue //Should this be a break?
			}
			elevatorSystems[networkMsg.Id] = networkMsg.ElevatorState
			msgCounter = networkMsg.Counter
			syncOrderSystem = Transition(elevatorId, networkMsg, syncOrderSystem)

			if elevatorSystems[elevatorId].Floor == -1 {
				log.Println("Elevator floor is -1, will not send to hra")
				continue //Should this be a break?
			}
			elevatorSystem := SyncOrderSystemToElevatorSystem(elevatorSystems, elevatorId, syncOrderSystem)
			updateHallLights(elevatorSystem.HallRequests)
			updateCabLights(elevatorSystem.ElevatorStates[elevatorId].CabRequests)
			hraOutput := hra.Decode(hra.AssignRequests(hra.Encode(elevatorSystem)))[elevatorId]
			if len(hraOutput) > 0 {
				select {
				case orderAssignment <- hraOutput:
				default:
					log.Println("No message sent")
				}
			} else {
				log.Println("Hra output empty, input to hra:", SyncOrderSystemToElevatorSystem(elevatorSystems, elevatorId, syncOrderSystem))
			}

		case peersUpdate := <-peersReciever: //This should edit syncOrderSystem or we need to pass around peerList
			if len(peersUpdate.Peers) == 0 {
				//Set to unknown
				log.Println("Reset to unknown")
				syncOrderSystem = NewSyncOrderSystem(elevatorId)
			}
			activePeers = peersUpdate.Peers
			syncOrderSystem = updateSyncOrderSystemFromPeerList(elevatorId, activePeers, syncOrderSystem)
			updatedElevatorSystems := make(map[string]ElevatorState)
			for _, peers := range activePeers {
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
			syncOrderSystem = RemoveOrder(elevatorId, orderToClear, syncOrderSystem)

		case <-timer.C: //Timer reset, send new state update
			msgCounter += 1
			networkTransmitter <- StateMsg{elevatorId, msgCounter, elevatorSystems[elevatorId], SyncOrderSystemToOrderSystem(elevatorId, syncOrderSystem)}
			timer.Reset(time.Millisecond * 10)
		}
	}
}

func updateSyncOrderSystemFromPeerList(localId string, peers []string, syncOrderSystem SyncOrderSystem) SyncOrderSystem {
	updatedSyncOrderSystem := NewSyncOrderSystem(localId)
	elevatorIds := append(peers, localId)
	for i, floor := range syncOrderSystem.HallRequests {
		for j, req := range floor {
			for id, order := range req {
				if contains(elevatorIds, id) {
					updatedSyncOrderSystem.HallRequests[i][j][id] = order
				}
			}
		}
	}
	for cabId, cabs := range syncOrderSystem.CabRequests {
		updatedSyncOrderSystem.CabRequests[cabId] = make([]SyncOrder, 4) //TODO do not hard code numbers
		for i, req := range cabs {
			for id, order := range req {
				if contains(elevatorIds, id) {
					updatedSyncOrderSystem.CabRequests[cabId][i] = make(SyncOrder)
					updatedSyncOrderSystem.CabRequests[cabId][i][id] = order
				}
			}
		}
	}
	return updatedSyncOrderSystem
}

func contains(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
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

func Transition(localId string, networkMsg StateMsg, updatedSyncOrderSystem SyncOrderSystem) SyncOrderSystem {
	updatedSyncOrderSystem = AddElevatorToSyncOrderSystem(localId, networkMsg, updatedSyncOrderSystem)

	orderSystem := SyncOrderSystemToOrderSystem(localId, updatedSyncOrderSystem)
	orderSystem.HallRequests = TransitionHallRequests(orderSystem.HallRequests, networkMsg.OrderSystem.HallRequests)

	_, ok := updatedSyncOrderSystem.CabRequests[networkMsg.Id]
	if ok && len(orderSystem.CabRequests[networkMsg.Id]) > 0 {
		orderSystem.CabRequests[localId] = TransitionCabRequests(orderSystem.CabRequests[localId], orderSystem.CabRequests[networkMsg.Id])
	} else {
		log.Println("Could not transition cabs. We did not add elevator", networkMsg.Id, "to syncOrderSystem")
	}
	updatedSyncOrderSystem = SystemToSyncOrderSystem(localId, updatedSyncOrderSystem, orderSystem)

	return ConsensusBarrierTransition(localId, updatedSyncOrderSystem)
}

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

func ConsensusBarrierTransition(localId string, OrderSystem SyncOrderSystem) SyncOrderSystem {
	//Transition all cabs
	{
		floor, newState := ConsensusTransitionSingleCab(localId, OrderSystem.CabRequests)
		for floor != -1 && newState != -1 { //Can this create nasty edge cases? Maybe have a validation test earlier to check for unknown orders
			OrderSystem.CabRequests[localId][floor][localId] = newState
			floor, newState = ConsensusTransitionSingleCab(localId, OrderSystem.CabRequests)
		}
	}

	//Transition all halls
	{
		floor, btn, newState := ConsensusTransitionSingleHall(localId, OrderSystem.HallRequests)
		for floor != -1 && btn != -1 {
			OrderSystem.HallRequests[floor][btn][localId] = newState
			floor, btn, newState = ConsensusTransitionSingleHall(localId, OrderSystem.HallRequests)
		}
	}
	return OrderSystem
}

// Cab and hall are very similar, we should refactor more
func ConsensusTransitionSingleCab(localId string, cabRequests map[string][]SyncOrder) (int, int) {
	for reqFloor, req := range cabRequests[localId] { //We only check our own cabs for consensus
		if AllValuesEqual(req) { //Consensus
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

func ConsensusTransitionSingleHall(localId string, hallRequests [][]SyncOrder) (int, int, int) { // (int, int ,int) is not clear, should have order type instead
	for reqFloor, row := range hallRequests {
		for reqBtn, req := range row {
			if AllValuesEqual(req) {
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

func NewSyncOrderSystem(initialKey string) SyncOrderSystem {
	// Initialize HallRequests with fixed sizes.
	initMap := SyncOrder{}
	initMap = make(map[string]int)
	initMap[initialKey] = unknownOrder // Init with unknown to just join NW
	hallRequests := [][]SyncOrder{{initMap, initMap}, {initMap, initMap}, {initMap, initMap}, {initMap, initMap}}
	for i := 0; i < 4; i++ { // Hard coded values FIX
		for j := 0; j < 2; j++ { // Hard coded values FIX
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

func SystemToSyncOrderSystem(localId string, syncOrderSystem SyncOrderSystem, orderSystem OrderSystem) SyncOrderSystem {
	for i, floor := range orderSystem.HallRequests {
		for j, req := range floor {
			syncOrderSystem.HallRequests[i][j][localId] = req
		}
	}
	//Should add other cabs aswell here
	for id, cabs := range syncOrderSystem.CabRequests { //THIS should be checked, should we nto loop through orderSyste here?
		for i, req := range cabs {
			syncOrderSystem.CabRequests[id][i][localId] = req[localId]
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
		newOrderSystem.CabRequests[id] = make([]int, 4) //TODO do not hard code numbers
		for i, req := range cabs {
			newOrderSystem.CabRequests[id][i] = req[localId] //This is dangrous but needed. See comments in AddElevatorToSyncOrderSystem. This could be the only place where we access this state this way.
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

func SyncOrderSystemToElevatorSystem(elevatorSystems map[string]ElevatorState, localId string, OrderSystem SyncOrderSystem) hra.ElevatorSystem {
	hraElevSys := hra.ElevatorSystem{
		HallRequests:   [][]int{{unknownOrder, unknownOrder}, {unknownOrder, unknownOrder}, {unknownOrder, unknownOrder}, {unknownOrder, unknownOrder}},
		ElevatorStates: map[string]hra.LocalElevatorState{},
	}

	//Fill halls
	for i, floor := range OrderSystem.HallRequests {
		for j, req := range floor {
			hraElevSys.HallRequests[i][j] = req[localId]
		}
	}
	//Loop through all IDs and add elevatorsystem pr id
	for id := range OrderSystem.CabRequests {
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
