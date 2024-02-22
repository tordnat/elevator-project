package musicplayer

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

var musicPid *os.Process

func PlayMusic(path string) {
	if path == "" {
		if musicPid != nil {
			musicPid.Signal(syscall.SIGCONT)
		}
		return
	}

	var cmd *exec.Cmd
	// Depending on the platform, you might need to adjust the paths and flags
	cmd = exec.Command("ffplay", path, "-autoexit", "-nodisp", "-hide_banner", "-loglevel", "warning", "-loop", "0")

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start music: %v", err)
		return
	}

	musicPid = cmd.Process
	log.Println("Playing elevator music")
}

func PauseMusic() {
	if musicPid != nil {
		musicPid.Signal(syscall.SIGSTOP)
		log.Println("Elevator music paused")
	}
}

func StopMusic() {
	if musicPid != nil {
		musicPid.Signal(syscall.SIGINT)
		log.Println("Elevator music stopped")
	}
}
