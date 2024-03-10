package requestSync_test

import (
	"elevator-project/requestSync"
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
