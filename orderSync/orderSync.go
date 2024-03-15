package orderSync

import (
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
	HallOrders [][]SyncOrder
	CabOrders  map[string][]SyncOrder
}
type OrderSystem struct {
	HallOrders [][]int
	CabOrders  map[string][]int
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

		case networkMsg := <-networkReciever:
			if networkMsg.Counter <= msgCounter && len(activePeers) != 1 && networkMsg.Id == localId { //To only listen to our own message when we are alone
				break
			}
			elevatorSystems[networkMsg.Id] = networkMsg.ElevatorState
			msgCounter = networkMsg.Counter

			syncOrderSystem = TransitionSystem(localId, networkMsg, syncOrderSystem, activePeers)

			if elevatorSystems[localId].Floor == -1 {
				log.Println("Elevator floor is -1, will not send to hra")
				break
			}

			elevatorSystem := SyncOrderSystemToElevatorSystem(elevatorSystems, localId, syncOrderSystem, activePeers)
			lights.UpdateHall(elevatorSystem.HallOrders)
			lights.UpdateCab(elevatorSystem.ElevatorStates[localId].CabOrders)
			hraOutput := hra.Decode(hra.AssignOrders(hra.Encode(elevatorSystem)))[localId]
			if len(hraOutput) > 0 {
				select {
				case orderAssignment <- hraOutput: //Non-blocking send
				default:
				}
			} else {
				log.Println("Hra output empty, input to hra:", elevatorSystem)
			}

		case peersUpdate := <-peersReciever:
			activePeers = peersUpdate.Peers

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
			timer.Reset(time.Millisecond * 5)
		}
	}
}

func AddOrder(localId string, syncOrderSystem SyncOrderSystem, btn elevio.ButtonEvent) SyncOrderSystem {
	if btn.Button == elevio.BT_Cab {
		syncOrderSystem.CabOrders[localId][btn.Floor][localId] = TransitionOrder(syncOrderSystem.CabOrders[localId][btn.Floor][localId], unconfirmedOrder)
	} else {
		syncOrderSystem.HallOrders[btn.Floor][btn.Button][localId] = TransitionOrder(syncOrderSystem.HallOrders[btn.Floor][btn.Button][localId], unconfirmedOrder)
	}
	return syncOrderSystem
}
func RemoveOrder(localId string, orderToClear orders.ClearFloorOrders, syncOrderSystem SyncOrderSystem) SyncOrderSystem {
	if orderToClear.Cab {
		syncOrderSystem.CabOrders[localId][orderToClear.Floor][localId] = TransitionOrder(syncOrderSystem.CabOrders[localId][orderToClear.Floor][localId], servicedOrder)
	}
	if orderToClear.HallUp {
		syncOrderSystem.HallOrders[orderToClear.Floor][0][localId] = TransitionOrder(syncOrderSystem.HallOrders[orderToClear.Floor][0][localId], servicedOrder)
	}
	if orderToClear.HallDown {
		syncOrderSystem.HallOrders[orderToClear.Floor][1][localId] = TransitionOrder(syncOrderSystem.HallOrders[orderToClear.Floor][1][localId], servicedOrder)
	}
	return syncOrderSystem
}

func TransitionSystem(localId string, networkMsg StateMsg, updatedSyncOrderSystem SyncOrderSystem, peers []string) SyncOrderSystem {
	updatedSyncOrderSystem = AddElevatorToSyncOrderSystem(localId, networkMsg, updatedSyncOrderSystem)

	return ConsensusBarrierTransition(localId, updatedSyncOrderSystem, peers)
}

func AddElevatorToSyncOrderSystem(localId string, networkMsg StateMsg, syncOrderSystem SyncOrderSystem) SyncOrderSystem {
	for floor, orders := range networkMsg.OrderSystem.HallOrders {
		for btn, networkorder := range orders {
			syncOrderSystem.HallOrders[floor][btn][localId] = TransitionOrder(syncOrderSystem.HallOrders[floor][btn][localId], networkorder)
			syncOrderSystem.HallOrders[floor][btn][networkMsg.Id] = networkorder
		}
	}
	for elevId, orders := range networkMsg.OrderSystem.CabOrders {
		_, ok := syncOrderSystem.CabOrders[elevId]
		if ok {
			for floor, order := range orders {
				for syncID := range networkMsg.OrderSystem.CabOrders {
					_, ok := syncOrderSystem.CabOrders[elevId][floor][syncID]
					if syncID == localId {
						if ok {
							syncOrderSystem.CabOrders[elevId][floor][syncID] = TransitionOrder(syncOrderSystem.CabOrders[elevId][floor][syncID], order)
						} else {
							syncOrderSystem.CabOrders[elevId][floor][syncID] = TransitionOrder(unknownOrder, order)
						}
					} else {
						syncOrderSystem.CabOrders[elevId][floor][syncID] = order
					}
				}
			}
		} else {
			syncOrderSystem.CabOrders[elevId] = make([]SyncOrder, elevator.N_FLOORS)
			for floor := 0; floor < elevator.N_FLOORS; floor++ {
				syncOrderSystem.CabOrders[elevId][floor] = make(SyncOrder)
			}
		}
	}
	return syncOrderSystem
}

func ConsensusBarrierTransition(localId string, syncOrderSystem SyncOrderSystem, peers []string) SyncOrderSystem {
	floor, newState := ConsensusTransitionSingleCab(localId, syncOrderSystem.CabOrders[localId], peers)
	if floor != -1 && newState != -1 {
		syncOrderSystem.CabOrders[localId][floor][localId] = newState
	}

	floor, btn, newState := ConsensusTransitionSingleHall(localId, syncOrderSystem.HallOrders, peers)
	if floor != -1 && btn != -1 {
		syncOrderSystem.HallOrders[floor][btn][localId] = newState
	}
	return syncOrderSystem
}

// Cab and hall are very similar, we should refactor more
func ConsensusTransitionSingleCab(localId string, CabOrders []SyncOrder, peers []string) (floor, order int) {
	for reqFloor, order := range CabOrders { //We only check our own cabs for consensus
		if ConsensusAmongPeers(order, peers) { //Consensus
			ourorder := order[localId]
			if ourorder == servicedOrder {
				return reqFloor, noOrder
			} else if ourorder == unconfirmedOrder {
				return reqFloor, confirmedOrder
			}
		}
	}
	return -1, -1
}

func ConsensusTransitionSingleHall(localId string, HallOrders [][]SyncOrder, peers []string) (floor, btn, order int) {
	for reqFloor, row := range HallOrders {
		for reqBtn, req := range row {
			if ConsensusAmongPeers(req, peers) {
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

func ConsensusAmongPeers(order SyncOrder, peers []string) bool {
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

func NewSyncOrderSystem(id string) SyncOrderSystem {
	HallOrders := make([][]SyncOrder, elevator.N_FLOORS)
	for i := 0; i < elevator.N_FLOORS; i++ {
		HallOrders[i] = make([]SyncOrder, elevator.N_HALL_BUTTONS)
		for j := 0; j < elevator.N_HALL_BUTTONS; j++ {
			initMap := make(SyncOrder)
			initMap[id] = unknownOrder // Init with unknown to just join network
			HallOrders[i][j] = map[string]int{id: unknownOrder}
		}
	}

	CabOrders := make(map[string][]SyncOrder)
	CabOrders[id] = make([]SyncOrder, elevator.N_FLOORS)
	for i := 0; i < elevator.N_FLOORS; i++ { //Fix
		CabOrders[id][i] = SyncOrder{id: unknownOrder}
	}
	return SyncOrderSystem{
		HallOrders: HallOrders,
		CabOrders:  CabOrders,
	}
}

func newOrderSystem(id string) OrderSystem {
	HallOrders := make([][]int, elevator.N_FLOORS)
	for i := 0; i < elevator.N_FLOORS; i++ {
		HallOrders[i] = make([]int, elevator.N_HALL_BUTTONS)
	}

	CabOrders := make(map[string][]int)
	CabOrders[id] = make([]int, elevator.N_FLOORS)
	for i := 0; i < elevator.N_FLOORS; i++ {
		CabOrders[id][i] = unknownOrder
	}

	return OrderSystem{
		HallOrders: HallOrders,
		CabOrders:  CabOrders,
	}
}

func UpdateSyncOrderSystem(localId string, syncOrderSystem SyncOrderSystem, orderSystem OrderSystem) SyncOrderSystem {
	for i, floor := range orderSystem.HallOrders {
		for j, req := range floor {
			syncOrderSystem.HallOrders[i][j][localId] = req
		}
	}
	for id, cabs := range orderSystem.CabOrders {
		for floor, req := range cabs {
			syncOrderSystem.CabOrders[id][floor][localId] = req
		}
	}
	return syncOrderSystem
}

func SyncOrderSystemToOrderSystem(localId string, syncOrderSystem SyncOrderSystem) OrderSystem {
	var newOrderSystem OrderSystem = newOrderSystem(localId)

	for i, floor := range syncOrderSystem.HallOrders {
		for j, req := range floor {
			newOrderSystem.HallOrders[i][j] = req[localId]
		}
	}
	for id, cabs := range syncOrderSystem.CabOrders {
		newOrderSystem.CabOrders[id] = make([]int, elevator.N_FLOORS)
		for i, req := range cabs {
			newOrderSystem.CabOrders[id][i] = req[localId]
		}
	}

	return newOrderSystem
}

func NewLocalElevatorState(elevatorSystem Elevator, id string, syncOrderSystem SyncOrderSystem) hra.HraLocalElevatorState {
	hraElevator := hra.HraLocalElevatorState{}
	switch elevatorSystem.Behaviour {
	case elevator.EB_Idle:
		hraElevator.Behaviour = "idle"
	case elevator.EB_DoorOpen:
		hraElevator.Behaviour = "doorOpen"
	case elevator.EB_Moving:
		hraElevator.Behaviour = "moving"
	}

	switch elevatorSystem.Direction {
	case elevio.MD_Stop:
		hraElevator.Direction = "stop"
	case elevio.MD_Up:
		hraElevator.Direction = "up"
	case elevio.MD_Down:
		hraElevator.Direction = "down"
	}
	hraElevator.Floor = elevatorSystem.Floor

	hraElevator.CabOrders = make([]bool, len(syncOrderSystem.CabOrders[id]))
	for i, req := range syncOrderSystem.CabOrders[id] {
		hraElevator.CabOrders[i] = (req[id] == confirmedOrder)
	}

	return hraElevator
}

func SyncOrderSystemToElevatorSystem(elevatorSystems map[string]Elevator, localId string, syncOrderSystem SyncOrderSystem, peers []string) hra.HraElevatorSystem {
	hraElevSys := hra.NewElevatorSystem(elevator.N_FLOORS)

	//Fill halls. Don't need peers, because we base it on our own id
	for i, floor := range syncOrderSystem.HallOrders {
		for j, req := range floor {
			hraElevSys.HallOrders[i][j] = (req[localId] == confirmedOrder)
		}
	}

	//We only want to add alive peers to HRA input
	for _, id := range peers {
		hraElevSys.ElevatorStates[id] = NewLocalElevatorState(elevatorSystems[id], id, syncOrderSystem)
	}

	return hraElevSys
}