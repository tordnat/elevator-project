package transition

const (
	unknownOrder = iota
	noOrder
	unconfirmedOrder
	confirmedOrder
	servicedOrder
)

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
