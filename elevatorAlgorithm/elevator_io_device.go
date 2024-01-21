package elevator 

import (
	"fmt"
	"net"
)

type Dirn enum { 
    D_Down  = -1,
    D_Stop  = 0,
    D_Up    = 1
}

type Button enum { 
    B_HallUp,
    B_HallDown,
    B_Cab
}


type ElevInputDevice struct {
    int (*floorSensor)(void);
    int (*requestButton)(int, Button);
    int (*stopButton)(void);
    int (*obstruction)(void);
    
}


type ElevOutputDevice struct {
    void (*floorIndicator)(int);
    void (*requestButtonLight)(int, Button, int);
    void (*doorLight)(int);
    void (*stopButtonLight)(int);
    void (*motorDirection)(Dirn);
}


attribute((constructor)) elev_init(void) Nil{
    elevator_hardware_init();
}

wrap_requestButton(int f, Button b) Nil{
    return elevator_hardware_get_button_signal(b, f);

}

wrap_requestButtonLight(int f, Button b, int v) Nil{
    elevator_hardware_set_button_lamp(b, f, v);
}

wrap_motorDirection(Dirn d) Nil{
    elevator_hardware_set_motor_direction(d);
}

elevio_getInputDevice() ElevInputDevice {
    return (ElevInputDevice){
        floorSensor    := &elevator_hardware_get_floor_sensor_signal,
        requestButton  := &_wrap_requestButton,
        stopButton     := &elevator_hardware_get_stop_signal,
        obstruction    := &elevator_hardware_get_obstruction_signal
    };
}

elevio_getOutputDevice() ElevOutputDevice {
    return (ElevOutputDevice){
        floorIndicator     := &elevator_hardware_set_floor_indicator,
        requestButtonLight := &_wrap_requestButtonLight,
        doorLight          := &elevator_hardware_set_door_open_lamp,
        stopButtonLight    := &elevator_hardware_set_stop_lamp,
        motorDirection     := &_wrap_motorDirection
    };
}

 elevio_dirn_toString(Dirn d) char* {
    if d == D_Up {
        return "D_Up" 
    } else if d == D_Down {
        return "D_Down"
    } return d == D_Stop{
        return "D_Stop"
    } else{
        return "D_UNDEFINED"
    }
}


elevio_button_toString(Button b) char* {
    if b == B_HallUp {
        return "B_HallUp" 
    } else if b == B_HallDown {
        return "B_HallDown"
    } return b == B_Cab{
        return "B_Cab"
    } else{
        return "B_UNDEFINED"
    }
}