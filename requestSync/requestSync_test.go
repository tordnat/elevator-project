package requestSync_test

import (
	"elevator-project/requestSync"
	"elevatorAlgorithm/elevator"
	"elevatorDriver/elevio"
	"fmt"
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
	if requestSync.TransitionOrder(currentOrder, networkOrder) != unconfirmedOrder {
		t.Error("Failed assert, did not transition to unconfirmedOrder")
	}
	if requestSync.TransitionOrder(unconfirmedOrder, unconfirmedOrder) != unconfirmedOrder {
		t.Error("Failed assert, did not transition to unconfirmedOrder")
	}
	if requestSync.TransitionOrder(unconfirmedOrder, confirmedOrder) != confirmedOrder {
		t.Error("Failed assert, did not transition to confirmedOrder")
	}
	if requestSync.TransitionOrder(unknownOrder, confirmedOrder) != confirmedOrder {
		t.Error("Failed assert, did not transition to confirmedOrder")
	}
	if requestSync.TransitionOrder(servicedOrder, confirmedOrder) != servicedOrder {
		t.Error("Failed assert, did not transition to servicedOrder")
	}
	if requestSync.TransitionOrder(servicedOrder, noOrder) != noOrder {
		t.Error("Failed assert, did not transition to noOrder")
	}
	if requestSync.TransitionOrder(unconfirmedOrder, servicedOrder) != servicedOrder {
		t.Error("Failed assert, did not transition to servicedOrder")
	}

	internalReq := []int{noOrder, unknownOrder, servicedOrder, noOrder}
	netReq := []int{noOrder, servicedOrder, unconfirmedOrder, confirmedOrder}
	result := []int{noOrder, servicedOrder, servicedOrder, confirmedOrder}
	if !areEqualArr(requestSync.TransitionCabRequests(internalReq, netReq), result) {
		t.Error("Failed assert, did not transition cabs correct")
		fmt.Println("Got: ", requestSync.TransitionCabRequests(internalReq, netReq))
		fmt.Println("Expected ", result)
	}
	internalReq2 := [][]int{
		{noOrder, unknownOrder, servicedOrder},
		{noOrder, unknownOrder, servicedOrder},
		{noOrder, unknownOrder, servicedOrder},
		{noOrder, unknownOrder, servicedOrder}}
	result2 := [][]int{
		{noOrder, unknownOrder, servicedOrder},
		{noOrder, unknownOrder, servicedOrder},
		{noOrder, unknownOrder, servicedOrder},
		{noOrder, unknownOrder, servicedOrder}}
	if !areEqualMat(requestSync.TransitionHallRequests(internalReq2, internalReq2), result2) {
		t.Error("Failed assert, did not transition halls correct")
		fmt.Println("Got: ", requestSync.TransitionHallRequests(internalReq2, internalReq2))
		fmt.Println("Expected ", result)
	}

}

func TestTransitionCabRequests(t *testing.T) {

	internalReq := []int{noOrder, unknownOrder, servicedOrder, noOrder}
	netReq := []int{noOrder, servicedOrder, unconfirmedOrder, confirmedOrder}
	result := []int{noOrder, servicedOrder, servicedOrder, confirmedOrder}
	if !areEqualArr(requestSync.TransitionCabRequests(internalReq, netReq), result) {
		t.Error("Failed assert, did not transition cabs correct")
		fmt.Println("Got: ", requestSync.TransitionCabRequests(internalReq, netReq))
		fmt.Println("Expected ", result)

		/*internalReq := []int{noOrder, unknownOrder, servicedOrder, noOrder}
		netReq := []int{noOrder, unknownOrder, servicedOrder, noOrder}
		if !areEqualArr(requestSync.TransitionCabRequests(internalReq, netReq), netReq) {
			t.Error("Failed assert, did not transition cabs correctly")
		}*/
	}

}

func TestConsensusBarrier(t *testing.T) {
	//Test consensus. These should be improved to check entire state, not just single orders
	localId := "0"
	elev1id := "1"
	elev2id := "2"
	peerList := []string{localId, elev1id, elev2id}
	orderSys := requestSync.NewSyncOrderSystem(localId)
	//Set floor zero cab req to unknown
	orderSys.CabRequests[localId][0][localId] = unconfirmedOrder
	orderSysAfterTrans := requestSync.ConsensusBarrierTransition(localId, orderSys, []string{localId})
	if orderSysAfterTrans.CabRequests[localId][0][localId] != confirmedOrder {
		t.Error("Failed assert, did not barrier transition cab correct")
	}
	if orderSysAfterTrans.CabRequests[localId][1][localId] != unknownOrder {
		t.Error("Failed assert, transitioned unknown order to", orderSysAfterTrans.CabRequests[localId][1][localId])
	}

	orderSys = requestSync.NewSyncOrderSystem(localId)
	//Set floor zero cab req to unknown
	orderSys.CabRequests[localId][0][localId] = confirmedOrder
	orderSysAfterTrans = requestSync.ConsensusBarrierTransition(localId, orderSys, []string{localId})
	if orderSysAfterTrans.CabRequests["0"][0]["0"] != confirmedOrder {
		t.Error("Failed assert, transitioned when we should have stayed")
	}

	orderSys = requestSync.NewSyncOrderSystem(localId)
	//Set floor zero cab req to unknown
	orderSys.CabRequests[localId][0][localId] = confirmedOrder

	elevatorSystem := requestSync.Elevator{elevator.EB_Idle, -1, elevio.MD_Stop}
	networkMsg := requestSync.StateMsg{localId, 2, elevatorSystem, requestSync.SyncOrderSystemToOrderSystem(localId, orderSys)}
	orderSysAfterTrans = requestSync.Transition(localId, networkMsg, orderSys, []string{localId})
	if orderSysAfterTrans.CabRequests[localId][0][localId] != confirmedOrder {
		t.Error("Failed assert, transitioned when we should have stayed")
	}

	// Test order completion
	orderSys = requestSync.NewSyncOrderSystem("0")
	orderSys.CabRequests["0"][0][localId] = servicedOrder
	orderSys.CabRequests["0"][0][elev1id] = servicedOrder
	orderSys.CabRequests["0"][0][elev2id] = servicedOrder

	orderSys.HallRequests[0][0][localId] = servicedOrder
	orderSys.HallRequests[0][0][elev1id] = servicedOrder
	orderSys.HallRequests[0][0][elev2id] = servicedOrder

	orderSysAfterTrans = requestSync.ConsensusBarrierTransition("0", orderSys, peerList)
	if orderSysAfterTrans.CabRequests["0"][0]["0"] != noOrder {
		t.Error("Cab order should be completed after transitioning got: ", orderSysAfterTrans.CabRequests["0"][0]["0"])
	}
	if orderSysAfterTrans.HallRequests[0][0]["0"] != noOrder {
		t.Error("Hall order should be completed after transitioning, got: ", orderSysAfterTrans.HallRequests[0][0]["0"])
	}
	if orderSysAfterTrans.CabRequests["0"][1]["0"] != unknownOrder {
		t.Error("Unknown cab got transtitioned")
	}

	orderSys = requestSync.NewSyncOrderSystem("0")
	orderSys.CabRequests["0"][0]["0"] = servicedOrder
	orderSysAfterTrans = requestSync.ConsensusBarrierTransition("0", orderSys, []string{"0"})
	if orderSysAfterTrans.CabRequests["0"][0]["0"] != noOrder {
		t.Error("Cab order should be completed after transitioning got: ", orderSysAfterTrans.CabRequests["1"][0]["0"])
	}
	if orderSysAfterTrans.CabRequests["0"][1]["0"] != unknownOrder {
		t.Error("Unknown cab got transtitioned")
	}
	//Test SystemToSyncOrderSystem
	syncOrderSys := requestSync.NewSyncOrderSystem("0")
	syncOrderSys.HallRequests[0][0][localId] = unconfirmedOrder
	syncOrderSys.HallRequests[0][0][elev1id] = unconfirmedOrder
	syncOrderSys.HallRequests[0][0][elev2id] = unconfirmedOrder
	normalOrderSys := requestSync.SyncOrderSystemToOrderSystem(localId, syncOrderSys)

	//All have unknown here
	newSys := requestSync.UpdateSyncOrderSystem(localId, syncOrderSys, normalOrderSys)
	if newSys.HallRequests[0][0][localId] != unconfirmedOrder && newSys.HallRequests[0][0][elev1id] != unconfirmedOrder && newSys.HallRequests[0][0][elev2id] != unconfirmedOrder {
		t.Error("Not unconfirmed")
	}
}

func areEqualArr(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func areEqualMat(a, b [][]int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}

	return true
}
