package lights

import "elevatorDriver/elevio"

func UpdateHall(hall_orders [][]int, lightOnCondition int) {
	for floor, floorRow := range hall_orders {
		for btn, order := range floorRow {
			elevio.SetButtonLamp(elevio.ButtonType(btn), floor, (order == lightOnCondition))
		}
	}
}

func UpdateCab(cab_orders []int, lightOnCondition int) {
	for floor, order := range cab_orders {
		elevio.SetButtonLamp(elevio.BT_Cab, floor, (order == lightOnCondition))
	}
}
