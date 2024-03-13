package orderSync

import (
	"elevator-project/transition"
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/hra"
	"elevatorAlgorithm/orders"
	"elevatorDriver/elevio"
	"elevatorDriver/lights"
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
	Hallorders [][]SyncOrder
	Caborders  map[string][]SyncOrder
}
type OrderSystem struct {
	Hallorders [][]int
	Caborders  map[string][]int
}

func Sync(elevatorSystemFromFSM chan elevator.Elevator, localId string, orderAssignment chan [][]bool, orderCompleted chan orders.ClearFloorOrders, peersReciever chan peers.PeerUpdate) {
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
			lights.UpdateHall(elevatorSystem.Hallorders, confirmedOrder)
			lights.UpdateCab(elevatorSystem.ElevatorStates[localId].Caborders, confirmedOrder)
			hraOutput := hra.Decode(hra.Assignorders(hra.Encode(elevatorSystem)))[localId]
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
			log.Println("Peers:", peersUpdate.Peers)
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
		syncOrderSystem.Caborders[localId][btn.Floor][localId] = transition.Order(syncOrderSystem.Caborders[localId][btn.Floor][localId], unconfirmedOrder)
	} else {
		syncOrderSystem.Hallorders[btn.Floor][btn.Button][localId] = transition.Order(syncOrderSystem.Hallorders[btn.Floor][btn.Button][localId], unconfirmedOrder)
	}
	return syncOrderSystem
}
func RemoveOrder(localId string, orderToClear orders.ClearFloorOrders, syncOrderSystem SyncOrderSystem) SyncOrderSystem {
	if orderToClear.Cab {
		syncOrderSystem.Caborders[localId][orderToClear.Floor][localId] = transition.Order(syncOrderSystem.Caborders[localId][orderToClear.Floor][localId], servicedOrder)
	}
	if orderToClear.HallUp {
		syncOrderSystem.Hallorders[orderToClear.Floor][0][localId] = transition.Order(syncOrderSystem.Hallorders[orderToClear.Floor][0][localId], servicedOrder)
	}
	if orderToClear.HallDown {
		syncOrderSystem.Hallorders[orderToClear.Floor][1][localId] = transition.Order(syncOrderSystem.Hallorders[orderToClear.Floor][1][localId], servicedOrder)
	}
	return syncOrderSystem
}

func Transition(localId string, networkMsg StateMsg, updatedSyncOrderSystem SyncOrderSystem, peers []string) SyncOrderSystem {
	updatedSyncOrderSystem = AddElevatorToSyncOrderSystem(localId, networkMsg, updatedSyncOrderSystem)

	orderSystem := SyncOrderSystemToOrderSystem(localId, updatedSyncOrderSystem)
	orderSystem.Hallorders = transition.Hall(orderSystem.Hallorders, networkMsg.OrderSystem.Hallorders)

	_, ok := updatedSyncOrderSystem.Caborders[networkMsg.Id]
	if ok && len(orderSystem.Caborders[networkMsg.Id]) > 0 {
		orderSystem.Caborders[localId] = transition.Cab(orderSystem.Caborders[localId], orderSystem.Caborders[networkMsg.Id])
	} else {
		log.Println("Could not transition cabs. We did not add elevator", networkMsg.Id, "to syncOrderSystem")
	}
	updatedSyncOrderSystem = UpdateSyncOrderSystem(localId, updatedSyncOrderSystem, orderSystem)

	return ConsensusBarrierTransition(localId, updatedSyncOrderSystem, peers)
}

func AddElevatorToSyncOrderSystem(localId string, networkMsg StateMsg, syncOrderSystem SyncOrderSystem) SyncOrderSystem {
	//Update our records of the view networkElevator has of our cabs
	for floor, networkorder := range networkMsg.OrderSystem.Caborders[localId] {
		syncOrderSystem.Caborders[localId][floor][networkMsg.Id] = networkorder
	}
	//Update our records of the view networkElevator has of halls
	for floor, orders := range networkMsg.OrderSystem.Hallorders {
		for btn, networkorder := range orders {
			syncOrderSystem.Hallorders[floor][btn][networkMsg.Id] = networkorder
		}
	}

	_, ok := syncOrderSystem.Caborders[networkMsg.Id]
	if !ok {
		syncOrderSystem.Caborders[networkMsg.Id] = make([]SyncOrder, len(syncOrderSystem.Caborders[localId]))
	}
	//Add/Update the cab orders of the other elevator into our own representation of them.
	//Because we only add to our own representation here, we could have just used int for this. We could do this because we never run a consensus transition on anyone elses cab orders.
	//This also means accessing syncOrderSystem.Caborders[networkMsg.Id][floor][NETWORKID] is always wrong
	for floor, req := range networkMsg.OrderSystem.Caborders[networkMsg.Id] {
		syncOrderSystem.Caborders[networkMsg.Id][floor] = make(SyncOrder)
		syncOrderSystem.Caborders[networkMsg.Id][floor][localId] = req
	}
	return syncOrderSystem
}

func ConsensusBarrierTransition(localId string, OrderSystem SyncOrderSystem, peers []string) SyncOrderSystem {
	floor, newState := ConsensusTransitionSingleCab(localId, OrderSystem.Caborders, peers)
	if floor != -1 && newState != -1 {
		OrderSystem.Caborders[localId][floor][localId] = newState
	}

	floor, btn, newState := ConsensusTransitionSingleHall(localId, OrderSystem.Hallorders, peers)
	if floor != -1 && btn != -1 {
		OrderSystem.Hallorders[floor][btn][localId] = newState
	}
	return OrderSystem
}

// Cab and hall are very similar, we should refactor more
func ConsensusTransitionSingleCab(localId string, caborders map[string][]SyncOrder, peers []string) (int, int) {
	for reqFloor, req := range caborders[localId] { //We only check our own cabs for consensus
		if AllPeersHaveSameValue(req, peers) { //Consensus
			ourorder := req[localId]
			if ourorder == servicedOrder {
				return reqFloor, noOrder
			} else if ourorder == unconfirmedOrder {
				return reqFloor, confirmedOrder
			}
		}
	}
	return -1, -1
}

func ConsensusTransitionSingleHall(localId string, hallorders [][]SyncOrder, peers []string) (int, int, int) { // (int, int ,int) is not clear, should have order type instead
	for reqFloor, row := range hallorders {
		for reqBtn, req := range row {
			if AllPeersHaveSameValue(req, peers) {
				ourorder := req[localId]
				if ourorder == servicedOrder {
					return reqFloor, reqBtn, noOrder
				} else if ourorder == unconfirmedOrder {
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
	hallorders := make([][]SyncOrder, elevator.N_FLOORS)
	for i := 0; i < elevator.N_FLOORS; i++ {
		hallorders[i] = make([]SyncOrder, elevator.N_HALL_BUTTONS)
		for j := 0; j < elevator.N_HALL_BUTTONS; j++ {
			initMap := make(SyncOrder)
			initMap[id] = unknownOrder // Init with unknown to just join network
			hallorders[i][j] = map[string]int{id: unknownOrder}
		}
	}

	caborders := make(map[string][]SyncOrder)
	caborders[id] = make([]SyncOrder, elevator.N_FLOORS)
	for i := 0; i < elevator.N_FLOORS; i++ { //Fix
		caborders[id][i] = SyncOrder{id: unknownOrder}
	}
	return SyncOrderSystem{
		Hallorders: hallorders,
		Caborders:  caborders,
	}
}

func newOrderSystem(id string) OrderSystem {
	hallorders := make([][]int, elevator.N_FLOORS)
	for i := 0; i < elevator.N_FLOORS; i++ {
		hallorders[i] = make([]int, elevator.N_HALL_BUTTONS)
	}

	caborders := make(map[string][]int)
	caborders[id] = make([]int, elevator.N_FLOORS)
	for i := 0; i < elevator.N_FLOORS; i++ {
		caborders[id][i] = unknownOrder
	}

	return OrderSystem{
		Hallorders: hallorders,
		Caborders:  caborders,
	}
}

func UpdateSyncOrderSystem(localId string, syncOrderSystem SyncOrderSystem, orderSystem OrderSystem) SyncOrderSystem {
	for i, floor := range orderSystem.Hallorders {
		for j, req := range floor {
			syncOrderSystem.Hallorders[i][j][localId] = req
		}
	}
	for id, cabs := range syncOrderSystem.Caborders {
		for floor, req := range cabs {
			syncOrderSystem.Caborders[id][floor][localId] = req[localId]
		}
	}
	return syncOrderSystem
}

func SyncOrderSystemToOrderSystem(localId string, syncOrderSystem SyncOrderSystem) OrderSystem {
	var newOrderSystem OrderSystem = newOrderSystem(localId)

	for i, floor := range syncOrderSystem.Hallorders {
		for j, req := range floor {
			newOrderSystem.Hallorders[i][j] = req[localId]
		}
	}
	for id, cabs := range syncOrderSystem.Caborders {
		newOrderSystem.Caborders[id] = make([]int, elevator.N_FLOORS)
		for i, req := range cabs {
			newOrderSystem.Caborders[id][i] = req[localId] //This is dangrous but needed. See comments in AddElevatorToSyncOrderSystem. This could be the only place where we access SyncOrderSystem, but only care about ourself (e.g we could use orderSystem all other places)
		}
	}

	return newOrderSystem
}

func NewLocalElevatorState(elevatorSystem Elevator, id string, OrderSystem SyncOrderSystem) hra.LocalElevatorState {
	localElevState := hra.LocalElevatorState{
		Behaviour: elevatorSystem.Behaviour,
		Floor:     elevatorSystem.Floor,
		Direction: elevatorSystem.Direction,
		Caborders: make([]int, len(OrderSystem.Caborders[id])),
	}

	for i, req := range OrderSystem.Caborders[id] {
		localElevState.Caborders[i] = req[id]
	}

	return localElevState
}

func SyncOrderSystemToElevatorSystem(elevatorSystems map[string]Elevator, localId string, OrderSystem SyncOrderSystem, peers []string) hra.ElevatorSystem {
	hraElevSys := hra.ElevatorSystem{
		Hallorders:     [][]int{{unknownOrder, unknownOrder}, {unknownOrder, unknownOrder}, {unknownOrder, unknownOrder}, {unknownOrder, unknownOrder}},
		ElevatorStates: map[string]hra.LocalElevatorState{},
	}

	//Fill halls. Don't need peers, because we base it on our own id
	for i, floor := range OrderSystem.Hallorders {
		for j, req := range floor {
			hraElevSys.Hallorders[i][j] = req[localId]
		}
	}

	//We only want to add alive peers to HRA input
	for _, id := range peers {
		hraElevSys.ElevatorStates[id] = NewLocalElevatorState(elevatorSystems[id], id, OrderSystem)
	}

	return hraElevSys
}
