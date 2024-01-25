package elevator

import (
	"elevatorDriver/elevio"
)

const N_FLOORS = 4
const N_BUTTONS = 3
const DOOR_OPEN_DURATION_S = 2

type ElevatorBehaviour int

const (
	EB_Idle = iota
	EB_DoorOpen
	EB_Moving
)

type Elevator struct {
	Floor     int
	Dirn      elevio.MotorDirection
	Requests  [N_FLOORS][N_BUTTONS]bool
	Behaviour ElevatorBehaviour
}

func NewElevator() Elevator {
	return Elevator{
		Floor:     -1,
		Dirn:      elevio.MD_Stop,
		Behaviour: EB_Idle,
	}
}

func eb_toString(eb ElevatorBehaviour) string {
	if eb == EB_Idle {
		return "EB_Idle"
	} else if eb == EB_DoorOpen {
		return "EB_DoorOpen"
	} else if eb == EB_Moving {
		return "EB_Moving"
	} else {
		return "EB_UNDEFINED"
	}
}

/*
func elevator_print(es Elevator) {
    fmt.Println("  +--------------------+")
    fmt.Println("  |floor = %-2d    |", es.floor)
    fmt.Println("  |dirn  = %-12.12s|", elevio_dirn_toString(es.dirn))
    fmt.Println("  |behav = %-12.12s|", eb_toString(es.behaviour))
    fmt.Println("  +--------------------+")
    fmt.Println("  |  | up⬆️ | dn ⬇️ | cab🛗|")
    for f := N_FLOORS-1; f >= 0; f-- {
        fmt.Print("  | %d", f)
        for btn := 0; btn < N_BUTTONS; btn++ {
            if (f == N_FLOORS-1 && btn == B_HallUp)  || (f == 0 && btn == B_HallDown){
                fmt.Print("|     ")
            } else {
                if fmt.Print(es.requests[f][btn]) {
                    fmt.Print("|  #  " , "|  -  ")
                }
            }
        }
        fmt.Println("|");
    }
    fmt.Println("  +--------------------+");
}

elevator_uninitialized() Elevator {
    return (Elevator){
        floor: -1,
        dirn: D_Stop,
        behaviour: EB_Idle,
        config: {
            clearRequestVariant: CV_All,
            doorOpenDuration_s: 3.0,
        },
    };
}
*/
