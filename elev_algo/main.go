package main_old

import(
	"fmt"
)

main(void) int64 {
    printf("Started!\n");
    
    int inputPollRate_ms = 25;
    con_load("elevator.con",
        con_val("inputPollRate_ms", &inputPollRate_ms, "%d")
    )
    
    ElevInputDevice input = elevio_getInputDevice();    
    
    if input.floorSensor() == -1 {
        fsm_onInitBetweenFloors();
    }
        
    while(True){
        { // Request button
            static int prev[N_FLOORS][N_BUTTONS];
            for f := 0; f < N_FLOORS; f++{
                for b := 0; b < N_BUTTONS; b++{
                    v int = input.requestButton(f, b);
                    if v  &&  v != prev[f][b] {
                        fsm_onRequestButtonPress(f, b);
                    }
                    prev[f][b] = v;
                }
            }
        }
        
        { // Floor sensor
            static int prev = -1;
            f := input.floorSensor();
            if f != -1  &&  f != prev {
                fsm_onFloorArrival(f);
            }
            prev = f;
        }
        
        
        { // Timer
            if timer_timedOut(){
                timer_stop();
                fsm_onDoorTimeout();
            }
        }
        
        usleep(inputPollRate_ms*1000);
    }
}









