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

func TestConsensusBarrier(t *testing.T) {
	//Test consensus. These should be improved to check entire state, not just single orders

	orderSys := requestSync.NewSyncOrderSystem("0")
	//Set floor zero cab req to unknown
	orderSys.CabRequests["0"][0]["0"] = unconfirmedOrder
	orderSysAfterTrans := requestSync.ConsensusBarrierTransition("0", orderSys)
	if orderSysAfterTrans.CabRequests["0"][0]["0"] != confirmedOrder {
		t.Error("Failed assert, did not barrier transition cab correct")
	}
	if orderSysAfterTrans.CabRequests["0"][1]["0"] != unknownOrder {
		t.Error("Failed assert, transitioned unknown order to", orderSysAfterTrans.CabRequests["0"][1]["0"])
	}

	orderSys = requestSync.NewSyncOrderSystem("0")
	//Set floor zero cab req to unknown
	orderSys.CabRequests["0"][0]["0"] = confirmedOrder
	orderSysAfterTrans = requestSync.ConsensusBarrierTransition("0", orderSys)
	if orderSysAfterTrans.CabRequests["0"][0]["0"] != confirmedOrder {
		t.Error("Failed assert, transitioned when we should have stayed")
	}

	orderSys = requestSync.NewSyncOrderSystem("0")
	//Set floor zero cab req to unknown
	orderSys.CabRequests["0"][0]["0"] = confirmedOrder

	elevatorSystem := requestSync.ElevatorState{elevator.EB_Idle, -1, elevio.MD_Stop}
	networkMsg := requestSync.StateMsg{"0", 2, elevatorSystem, requestSync.SyncSystemToOrderSystem("0", orderSys)}

	orderSysAfterTrans = requestSync.Transition("0", networkMsg, orderSys)
	if orderSysAfterTrans.CabRequests["0"][0]["0"] != confirmedOrder {
		t.Error("Failed assert, transitioned when we should have stayed")
	}

	// Test order completion
	orderSys = requestSync.NewSyncOrderSystem("0")
	orderSys.CabRequests["0"][0]["0"] = servicedOrder
	orderSys.CabRequests["0"][0]["1"] = servicedOrder
	orderSysAfterTrans = requestSync.ConsensusBarrierTransition("0", orderSys)
	if orderSysAfterTrans.CabRequests["0"][0]["0"] != noOrder {
		t.Error("Cab order should be completed after transitioning got: ", orderSysAfterTrans.CabRequests["1"][0]["0"])
	}
	if orderSysAfterTrans.CabRequests["0"][1]["0"] != unknownOrder {
		t.Error("Unknown cab got transtitioned")
	}

	orderSys = requestSync.NewSyncOrderSystem("0")
	orderSys.CabRequests["0"][0]["0"] = servicedOrder
	orderSysAfterTrans = requestSync.ConsensusBarrierTransition("0", orderSys)
	if orderSysAfterTrans.CabRequests["0"][0]["0"] != noOrder {
		t.Error("Cab order should be completed after transitioning got: ", orderSysAfterTrans.CabRequests["1"][0]["0"])
	}
	if orderSysAfterTrans.CabRequests["0"][1]["0"] != unknownOrder {
		t.Error("Unknown cab got transtitioned")
	}

}

func TestAddElevatorToSyncOrderSystem(t *testing.T) {
	orderSys := requestSync.NewSyncOrderSystem("0")
	elevtorMsg := requestSync.StateMsg{"1", 2, requestSync.ElevatorState{elevator.EB_Idle, -1, elevio.MD_Stop}, requestSync.SyncSystemToOrderSystem("1", orderSys)}
	orderSys = requestSync.AddElevatorToSyncOrderSystem("1", elevtorMsg, orderSys)

	if orderSys.CabRequests["1"] != elevtorMsg.OrderSystem.CabRequests["1"] {
		t.Error("Failed assert, elevator not added")
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
