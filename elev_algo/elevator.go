import(
    "fmt"

)

type ElevatorBehaviour enum {
    EB_Idle,
    EB_DoorOpen,
    EB_Moving
}

type ClearRequestVariant enum {
    // Assume everyone waiting for the elevator gets on the elevator, even if 
    // they will be traveling in the "wrong" direction for a while
    CV_All,
    
    // Assume that only those that want to travel in the current direction 
    // enter the elevator, and keep waiting outside otherwise
    CV_InDirn,
}

type Elevator struct {
    floor int;
    dirn Dirn;
    requests[N_FLOORS][N_BUTTONS] int;
    behaviour ElevatorBehaviour;
    
    struct {
        clearRequestVariant ClearRequestVariant;
        doorOpenDuration_s double;
    } config;    
}

eb_toString(eb ElevatorBehaviour) char* {
    if eb == EB_Idle {
        return "EB_Idle"
    } else if eb == EB_DoorOpen  {
        return "EB_DoorOpen"
    } return eb == EB_Moving{
        return "EB_Moving"
    } else{
        return "EB_UNDEFINED"
    }
}

elevator_print(es Elevator) Nil {
    fmt.Print("  +--------------------+\n");
    fmt.Print(
        "  |floor = %-2d          |\n"
        "  |dirn  = %-12.12s|\n"
        "  |behav = %-12.12s|\n",
        es.floor,
        elevio_dirn_toString(es.dirn),
        eb_toString(es.behaviour)
    );
    fmt.Print("  +--------------------+\n");
    fmt.Print("  |  | up  | dn  | cab |\n");
    for f := N_FLOORS-1; f >= 0; f-- {
        fmt.Print("  | %d", f);
        for btn := 0; btn < N_BUTTONS; btn++ {
            if (f == N_FLOORS-1 && btn == B_HallUp)  || (f == 0 && btn == B_HallDown){
                fmt.Print("|     ");
            } else {
                fmt.Print(es.requests[f][btn] ? "|  #  " : "|  -  ");
            }
        }
        fmt.Print("|\n");
    }
    fmt.Print("  +--------------------+\n");
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