package transition

const (
	unknownOrder = iota
	noOrder
	unconfirmedOrder
	confirmedOrder
	servicedOrder
)

// These are very similar to the hraHallRequestTypeToBool and hraCabRequestTypeToBool. Consider merging them and passing modifier function
func Cab(internalRequests []int, networkRequests []int) []int {
	for i, req := range internalRequests {
		internalRequests[i] = Order(req, networkRequests[i])
	}
	return internalRequests
}

func Hall(internalRequests [][]int, networkRequests [][]int) [][]int {
	for i, row := range internalRequests {
		for j, req := range row {
			internalRequests[i][j] = Order(req, networkRequests[i][j])
		}
	}
	return internalRequests
}

func Order(currentOrder int, updatedOrder int) int {
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
