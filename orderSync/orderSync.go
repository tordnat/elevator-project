package orderSync

import (
	"elevatorControl/elevator"
	"elevatorControl/hra"
	"elevatorControl/orders"
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
	OrderSystem   NetworkOrderSystem
}

const (
	unknownStatus = iota
	noOrder
	unconfirmedOrder
	confirmedOrder
	servicedOrder
)

const bcastPort int = 25565

type SyncOrder map[string]int //Map of what each elevator thinks the state of this order is

type SyncOrderSystem struct {
	HallOrders [][]SyncOrder
	CabOrders  map[string][]SyncOrder
}
type NetworkOrderSystem struct {
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

	timer := time.NewTimer(10 * time.Millisecond)
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

		case <-timer.C: 
			msgCounter += 1
			networkTransmitter <- StateMsg{localId, msgCounter, elevatorSystems[localId], SyncOrderSystemToNetworkOrderSystem(localId, syncOrderSystem)}
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
	for elevId, cabOrders := range networkMsg.OrderSystem.CabOrders {
		if _, exists := syncOrderSystem.CabOrders[elevId]; !exists {
			syncOrderSystem = initializeCabOrdersForElevator(elevId, syncOrderSystem)
		}
		for floor, order := range cabOrders {
			syncOrderSystem = syncCabOrderFloor(elevId, localId, floor, order, networkMsg, syncOrderSystem)
		}
	}
	return syncOrderSystem
}
func initializeCabOrdersForElevator(elevId string, syncOrderSystem SyncOrderSystem) SyncOrderSystem {
	syncOrderSystem.CabOrders[elevId] = make([]SyncOrder, elevator.N_FLOORS)
	for floor := 0; floor < elevator.N_FLOORS; floor++ {
		syncOrderSystem.CabOrders[elevId][floor] = make(SyncOrder)
	}
	return syncOrderSystem
}
func syncCabOrderFloor(elevId string, localId string, floor int, order int, networkMsg StateMsg, syncOrderSystem SyncOrderSystem) SyncOrderSystem {
	for syncID := range networkMsg.OrderSystem.CabOrders {
		currentOrder, exists := syncOrderSystem.CabOrders[elevId][floor][syncID]
		if syncID == localId {
			if exists {
				syncOrderSystem.CabOrders[elevId][floor][syncID] = TransitionOrder(currentOrder, order)
			} else {
				syncOrderSystem.CabOrders[elevId][floor][syncID] = TransitionOrder(unknownStatus, order)
			}
		} else {
			syncOrderSystem.CabOrders[elevId][floor][syncID] = order
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

func ConsensusTransitionSingleCab(localId string, CabOrders []SyncOrder, peers []string) (floor, order int) {
	for orderFloor, order := range CabOrders {
		if ConsensusAmongPeers(order, peers) {
			ourorder := order[localId]
			if ourorder == servicedOrder {
				return orderFloor, noOrder
			} else if ourorder == unconfirmedOrder {
				return orderFloor, confirmedOrder
			}
		}
	}
	return -1, -1
}

func ConsensusTransitionSingleHall(localId string, HallOrders [][]SyncOrder, peers []string) (floor, btn, order int) {
	for orderFloor, row := range HallOrders {
		for orderBtn, order := range row {
			if ConsensusAmongPeers(order, peers) {
				ourorder := order[localId]
				if ourorder == servicedOrder {
					return orderFloor, orderBtn, noOrder
				} else if ourorder == unconfirmedOrder {
					return orderFloor, orderBtn, confirmedOrder
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
	if currentOrder == unknownStatus { //Catch up if we just joined
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
			initMap[id] = unknownStatus
			HallOrders[i][j] = map[string]int{id: unknownStatus}
		}
	}

	CabOrders := make(map[string][]SyncOrder)
	CabOrders[id] = make([]SyncOrder, elevator.N_FLOORS)
	for i := 0; i < elevator.N_FLOORS; i++ {
		CabOrders[id][i] = SyncOrder{id: unknownStatus}
	}
	return SyncOrderSystem{
		HallOrders: HallOrders,
		CabOrders:  CabOrders,
	}
}

func newOrderSystem(id string) NetworkOrderSystem {
	HallOrders := make([][]int, elevator.N_FLOORS)
	for i := 0; i < elevator.N_FLOORS; i++ {
		HallOrders[i] = make([]int, elevator.N_HALL_BUTTONS)
	}

	CabOrders := make(map[string][]int)
	CabOrders[id] = make([]int, elevator.N_FLOORS)
	for i := 0; i < elevator.N_FLOORS; i++ {
		CabOrders[id][i] = unknownStatus
	}

	return NetworkOrderSystem{
		HallOrders: HallOrders,
		CabOrders:  CabOrders,
	}
}

func UpdateSyncOrderSystem(localId string, syncOrderSystem SyncOrderSystem, orderSystem NetworkOrderSystem) SyncOrderSystem {
	for i, floor := range orderSystem.HallOrders {
		for j, order := range floor {
			syncOrderSystem.HallOrders[i][j][localId] = order
		}
	}
	for id, cabs := range orderSystem.CabOrders {
		for floor, order := range cabs {
			syncOrderSystem.CabOrders[id][floor][localId] = order
		}
	}
	return syncOrderSystem
}

func SyncOrderSystemToNetworkOrderSystem(localId string, syncOrderSystem SyncOrderSystem) NetworkOrderSystem {
	var newOrderSystem NetworkOrderSystem = newOrderSystem(localId)

	for i, floorOrder := range syncOrderSystem.HallOrders {
		for j, order := range floorOrder {
			newOrderSystem.HallOrders[i][j] = order[localId]
		}
	}
	for id, cabOrders := range syncOrderSystem.CabOrders {
		newOrderSystem.CabOrders[id] = make([]int, elevator.N_FLOORS)
		for i, order := range cabOrders {
			newOrderSystem.CabOrders[id][i] = order[localId]
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
	for i, order := range syncOrderSystem.CabOrders[id] {
		hraElevator.CabOrders[i] = (order[id] == confirmedOrder)
	}

	return hraElevator
}

func SyncOrderSystemToElevatorSystem(elevatorSystems map[string]Elevator, localId string, syncOrderSystem SyncOrderSystem, peers []string) hra.HraElevatorSystem {
	hraElevSys := hra.NewElevatorSystem(elevator.N_FLOORS)

	for i, floor := range syncOrderSystem.HallOrders {
		for j, order := range floor {
			hraElevSys.HallOrders[i][j] = (order[localId] == confirmedOrder)
		}
	}

	//We only want to add alive peers to HRA input
	for _, id := range peers {
		hraElevSys.ElevatorStates[id] = NewLocalElevatorState(elevatorSystems[id], id, syncOrderSystem)
	}

	return hraElevSys
}
