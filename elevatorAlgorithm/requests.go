package requests

type DirnBehaviourPair struct {
    Dirn                dirn;
    ElevatorBehaviour   behaviour;
} DirnBehaviourPair;

func requests_above(e Elevator) bool{
	for f := e.floor+1; f < N_FLOORS; f++{
		for btn := 0; btn < N_BUTTONS; b++{
			if(e.rquests[f][btn]){
				return True;
			}
		}
	}
	return False;
}

func requests_below(e Elevator) bool{
	for f := 0; f < e.floor; f++{
		for btn := 0; btn < N_BUTTONS; b++{
			if(e.requests[f][btn]){
                return True;
            }
		}
	}
	return False;
}

func requests_here(e Elevator) bool{
	for btn := 0; btn < N_BUTTONS; btn++{
		if(e.requests[e.floor][btn]){
            return True;
        }
	}
	return False;
}

func requests_chooseDirection(e Elevator) DirnBehaviourPair{
	switch e.dirn{
	case D_up:
		if requests_above(e){
			return DirnBehaviourPair{D_Up,   EB_Moving}
		} else if requests_here(e){
			return DirnBehaviourPair{D_Down, EB_DoorOpen}
		} else if requests_below(e){
			return DirnBehaviourPair{D_Down, EB_Moving} 
		} else{
			return DirnBehaviourPair{D_Stop, EB_Idle}
		}
	case D_Down:
        if requestsBelow(e) {
            return DirnBehaviourPair{D_Down, EB_Moving}
        } else if requestsHere(e) {
            return DirnBehaviourPair{D_Up, EB_DoorOpen}
        } else if requestsAbove(e) {
            return DirnBehaviourPair{D_Up, EB_Moving}
        } else {
            return DirnBehaviourPair{D_Stop, EB_Idle}
        }
    case D_Stop:
        if requestsHere(e) {
            return DirnBehaviourPair{D_Stop, EB_DoorOpen}
        } else if requestsAbove(e) {
            return DirnBehaviourPair{D_Up, EB_Moving}
        } else if requestsBelow(e) {
            return DirnBehaviourPair{D_Down, EB_Moving}
        } else {
            return DirnBehaviourPair{D_Stop, EB_Idle}
        }
    default:
        return DirnBehaviourPair{D_Stop, EB_Idle}

	}
}

func requests_shouldStop(e Elevator){
	switch e.dirn{
	case D_Down:
		return e.requests[e.floor][B_HallDown] || e.requests[e.floor][B_Cab] || !requests_below(e);
	case D_Stop:
		return e.requests[e.floor][B_HallUp] || e.requests[e.floor][B_Cab] || !requests_above(e);
	case D_Stop:
    default:
        return 1;
	}
}

func requests_shouldClearImmediately(e Elevator, btn_floor int, btn_type Button) bool{
	switch e.config.clearRequestVariant{
    case CV_All:
        return e.floor == btn_floor;
    case CV_InDirn:
        return 
            e.floor == btn_floor && 
            (
                (e.dirn == D_Up   && btn_type == B_HallUp)    ||
                (e.dirn == D_Down && btn_type == B_HallDown)  ||
                e.dirn == D_Stop ||
                btn_type == B_Cab
            );  
    default:
        return 0;
    }
}

func requests_clearAtCurrentFloor(e Elevator){
	switch e.config.clearRequestVariant{
    case CV_All:
        for btn Button = 0; btn < N_BUTTONS; btn++ {
            e.requests[e.floor][btn] = 0;
        }
        break;
        
    case CV_InDirn:
        e.requests[e.floor][B_Cab] = 0;
        switch e.dirn{
        case D_Up:
            if !requests_above(e) && !e.requests[e.floor][B_HallUp] {
                e.requests[e.floor][B_HallDown] = 0;
            }
            e.requests[e.floor][B_HallUp] = 0;
            break;
            
        case D_Down:
            if !requests_below(e) && !e.requests[e.floor][B_HallDown] {
                e.requests[e.floor][B_HallUp] = 0;
            }
            e.requests[e.floor][B_HallDown] = 0;
            break;
            
        case D_Stop:
        default:
            e.requests[e.floor][B_HallUp] = 0;
            e.requests[e.floor][B_HallDown] = 0;
            break;
        }
        break;
        
    default:
        break;
    }
    
    return e;
}
