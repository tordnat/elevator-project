package orderSync_test

import (
	"elevator-project/orderSync"
	"elevatorAlgorithm/elevator"
	"elevatorDriver/elevio"
	"testing"
)

const (
	unknownOrder = iota
	noOrder
	unconfirmedOrder
	confirmedOrder
	servicedOrder
)

func TestTransitionOrder(t *testing.T) {
	currentOrder := noOrder
	networkOrder := unconfirmedOrder
	if orderSync.TransitionOrder(currentOrder, networkOrder) != unconfirmedOrder {
		t.Error("Failed assert, did not transition to unconfirmedOrder")
	}
	if orderSync.TransitionOrder(unconfirmedOrder, unconfirmedOrder) != unconfirmedOrder {
		t.Error("Failed assert, did not transition to unconfirmedOrder")
	}
	if orderSync.TransitionOrder(unconfirmedOrder, confirmedOrder) != confirmedOrder {
		t.Error("Failed assert, did not transition to confirmedOrder")
	}
	if orderSync.TransitionOrder(unknownOrder, confirmedOrder) != confirmedOrder {
		t.Error("Failed assert, did not transition to confirmedOrder")
	}
	if orderSync.TransitionOrder(servicedOrder, confirmedOrder) != servicedOrder {
		t.Error("Failed assert, did not transition to servicedOrder")
	}
	if orderSync.TransitionOrder(servicedOrder, noOrder) != noOrder {
		t.Error("Failed assert, did not transition to noOrder")
	}
	if orderSync.TransitionOrder(unconfirmedOrder, servicedOrder) != servicedOrder {
		t.Error("Failed assert, did not transition to servicedOrder")
	}
}

func TestConsensusBarrier(t *testing.T) {
	//Test consensus. These should be improved to check entire state, not just single orders
	localId := "0"
	elev1id := "1"
	elev2id := "2"
	peerList := []string{localId, elev1id, elev2id}
	orderSys := orderSync.NewSyncOrderSystem(localId)
	//Set floor zero cab order to unknown
	orderSys.CabOrders[localId][0][localId] = unconfirmedOrder
	orderSysAfterTrans := orderSync.ConsensusBarrierTransition(localId, orderSys, []string{localId})
	if orderSysAfterTrans.CabOrders[localId][0][localId] != confirmedOrder {
		t.Error("Failed assert, did not barrier transition cab correct")
	}
	if orderSysAfterTrans.CabOrders[localId][1][localId] != unknownOrder {
		t.Error("Failed assert, transitioned unknown order to", orderSysAfterTrans.CabOrders[localId][1][localId])
	}

	orderSys = orderSync.NewSyncOrderSystem(localId)
	//Set floor zero cab order to unknown
	orderSys.CabOrders[localId][0][localId] = confirmedOrder
	orderSysAfterTrans = orderSync.ConsensusBarrierTransition(localId, orderSys, []string{localId})
	if orderSysAfterTrans.CabOrders["0"][0]["0"] != confirmedOrder {
		t.Error("Failed assert, transitioned when we should have stayed")
	}

	orderSys = orderSync.NewSyncOrderSystem(localId)
	//Set floor zero cab order to unknown
	orderSys.CabOrders[localId][0][localId] = confirmedOrder

	elevatorSystem := orderSync.Elevator{elevator.EB_Idle, -1, elevio.MD_Stop}
	networkMsg := orderSync.StateMsg{localId, 2, elevatorSystem, orderSync.SyncOrderSystemToNetworkOrderSystem(localId, orderSys)}
	orderSysAfterTrans = orderSync.TransitionSystem(localId, networkMsg, orderSys, []string{localId})

	if orderSysAfterTrans.CabOrders[localId][0][localId] != confirmedOrder {
		t.Error("Failed assert, transitioned when we should have stayed")
	}

	// Test order completion
	orderSys = orderSync.NewSyncOrderSystem("0")
	orderSys.CabOrders["0"][0][localId] = servicedOrder
	orderSys.CabOrders["0"][0][elev1id] = servicedOrder
	orderSys.CabOrders["0"][0][elev2id] = servicedOrder

	orderSys.HallOrders[0][0][localId] = servicedOrder
	orderSys.HallOrders[0][0][elev1id] = servicedOrder
	orderSys.HallOrders[0][0][elev2id] = servicedOrder

	orderSysAfterTrans = orderSync.ConsensusBarrierTransition("0", orderSys, peerList)
	if orderSysAfterTrans.CabOrders["0"][0]["0"] != noOrder {
		t.Error("Cab order should be completed after transitioning got: ", orderSysAfterTrans.CabOrders["0"][0]["0"])
	}
	if orderSysAfterTrans.HallOrders[0][0]["0"] != noOrder {
		t.Error("Hall order should be completed after transitioning, got: ", orderSysAfterTrans.HallOrders[0][0]["0"])
	}
	if orderSysAfterTrans.CabOrders["0"][1]["0"] != unknownOrder {
		t.Error("Unknown cab got transtitioned")
	}

	orderSys = orderSync.NewSyncOrderSystem("0")
	orderSys.CabOrders["0"][0]["0"] = servicedOrder
	orderSysAfterTrans = orderSync.ConsensusBarrierTransition("0", orderSys, []string{"0"})
	if orderSysAfterTrans.CabOrders["0"][0]["0"] != noOrder {
		t.Error("Cab order should be completed after transitioning got: ", orderSysAfterTrans.CabOrders["1"][0]["0"])
	}
	if orderSysAfterTrans.CabOrders["0"][1]["0"] != unknownOrder {
		t.Error("Unknown cab got transtitioned")
	}
	//Test SystemToSyncOrderSystem
	syncOrderSys := orderSync.NewSyncOrderSystem("0")
	syncOrderSys.HallOrders[0][0][localId] = unconfirmedOrder
	syncOrderSys.HallOrders[0][0][elev1id] = unconfirmedOrder
	syncOrderSys.HallOrders[0][0][elev2id] = unconfirmedOrder
	normalOrderSys := orderSync.SyncOrderSystemToNetworkOrderSystem(localId, syncOrderSys)

	//All have unknown here
	newSys := orderSync.UpdateSyncOrderSystem(localId, syncOrderSys, normalOrderSys)
	if newSys.HallOrders[0][0][localId] != unconfirmedOrder && newSys.HallOrders[0][0][elev1id] != unconfirmedOrder && newSys.HallOrders[0][0][elev2id] != unconfirmedOrder {
		t.Error("Not unconfirmed")
	}
}
