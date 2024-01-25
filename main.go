package main

import (
	"elevatorAlgorithm/elevator"
	"elevatorAlgorithm/fsm"
	"elevatorAlgorithm/timer"
	"elevatorDriver/elevio"
	"log/slog"
	"os"
)

func main() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: false,
	})
	logger := slog.New(handler)

	slog.SetDefault(logger)

	slog.Info("Elevator starting 🛗")
	elevio.Init("localhost:15657", elevator.N_FLOORS)
	if elevio.GetFloor() == -1 {
		fsm.OnInitBetweenFloors()
	}
	prevFloor := -1
	var prev_btn [elevator.N_FLOORS][elevator.N_BUTTONS]bool

	for {
		{ //Request button
			for f := 0; f < elevator.N_FLOORS; f++ {
				for b := 0; b < elevator.N_BUTTONS; b++ {
					v := elevio.GetButton(elevio.ButtonType(b), f)
					if v && v != prev_btn[f][b] {
						fsm.OnRequestButtonPress(f, elevio.ButtonType(b))
					}
					prev_btn[f][b] = v
				}
			}
		}
		{ //Floor sensor
			f := elevio.GetFloor()
			if f != -1 && f != prevFloor {
				fsm.OnFloorArrival(f)
			}
			prevFloor = f
		}
		{ // Timer
			if timer.TimerActive {
				fsm.OnDoorTimeOut()
			}
		}
	}

}
