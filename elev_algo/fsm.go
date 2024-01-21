package fsm

import(
	"fmt"
)

var(
	elevator Elevator;
	outputDevice ElevOutputDevice;
)

attribute(constructor) fsm_init() static void{
    elevator := elevator_uninitialized();
    
    con_load("elevator.con",
        con_val("doorOpenDuration_s", &elevator.config.doorOpenDuration_s, "%lf")
        con_enum("clearRequestVariant", &elevator.config.clearRequestVariant,
            con_match(CV_All)
            con_match(CV_InDirn)
        )
    )
    
    outputDevice = elevio_getOutputDevice();

}

setAllLights(es Elevator) Nil{
    for floor := 0; floor < N_FLOORS; floor++ {
        for btn := 0; btn < N_BUTTONS; btn++ {
            outputDevice.requestButtonLight(floor, btn, es.requests[floor][btn]);
        }
    }
}

fsm_onInitBetweenFloors() Nil{
    outputDevice.motorDirection(D_Down);
    elevator.dirn = D_Down;
    elevator.behaviour = EB_Moving;
}

fsm_onRequestButtonPress(btn_floor int64, btn_type Button) Nil {
	fmt.Print("\n\n%s(%d, %s)\n", __FUNCTION__, btn_floor, elevio_button_toString(btn_type));
    elevator_print(elevator);

	switch elevator.behaviour{
	case EB_DoorOpen:
        if requests_shouldClearImmediately(elevator, btn_floor, btn_type) {
            timer_start(elevator.config.doorOpenDuration_s);
        } else {
            elevator.requests[btn_floor][btn_type] = 1;
        }
        break;

    case EB_Moving:
        elevator.requests[btn_floor][btn_type] = 1;
        break;
        
    case EB_Idle:    
        elevator.requests[btn_floor][btn_type] = 1;
        pair DirnBehaviourPair = requests_chooseDirection(elevator);
        elevator.dirn = pair.dirn;
        elevator.behaviour = pair.behaviour;
        switch pair.behaviour {
        case EB_DoorOpen:
            outputDevice.doorLight(1);
            timer_start(elevator.config.doorOpenDuration_s);
            elevator = requests_clearAtCurrentFloor(elevator);
            break;

        case EB_Moving:
            outputDevice.motorDirection(elevator.dirn);
            break;
            
        case EB_Idle:
            break;
        }
        break;
	}

	setAllLights(elevator);
    
    pfmt.Print("\nNew state:\n");
    elevator_print(elevator);
}

fsm_onFloorArrival(int newFloor) Nil {
    fmt.Print("\n\n%s(%d)\n", __FUNCTION__, newFloor);
    elevator_print(elevator);
    
    elevator.floor = newFloor;
    
    outputDevice.floorIndicator(elevator.floor);
    
    switch elevator.behaviour{
    case EB_Moving:
        if requests_shouldStop(elevator) {
            outputDevice.motorDirection(D_Stop);
            outputDevice.doorLight(1);
            elevator = requests_clearAtCurrentFloor(elevator);
            timer_start(elevator.config.doorOpenDuration_s);
            setAllLights(elevator);
            elevator.behaviour = EB_DoorOpen;
        }
        break;
    default:
        break;
    }
    
    fmt.Print("\nNew state:\n");
    elevator_print(elevator); 
}




fsm_onDoorTimeout(void) Nil {
    fmt.Print("\n\n%s()\n", __FUNCTION__);
    elevator_print(elevator);
    
    switch elevator.behaviour {
    case EB_DoorOpen:;
        DirnBehaviourPair pair = requests_chooseDirection(elevator);
        elevator.dirn = pair.dirn;
        elevator.behaviour = pair.behaviour;
        
        switch elevator.behaviour {
        case EB_DoorOpen:
            timer_start(elevator.config.doorOpenDuration_s);
            elevator = requests_clearAtCurrentFloor(elevator);
            setAllLights(elevator);
            break;
        case EB_Moving:
        case EB_Idle:
            outputDevice.doorLight(0);
            outputDevice.motorDirection(elevator.dirn);
            break;
        }
        break;
    default:
        break;
    }
    
    fmt.Print("\nNew state:\n");
    elevator_print(elevator);
}