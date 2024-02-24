package musicplayer

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

var musicPid *os.Process

func PlayMusic(path string, loop_music bool) {
	if path == "" {
		if musicPid != nil {
			musicPid.Signal(syscall.SIGCONT)
		}
		return
	}

	var cmd *exec.Cmd

	if loop_music {
		cmd = exec.Command("ffplay", path, "-autoexit", "-nodisp", "-hide_banner", "-loglevel", "warning", "-loop", "0")
	} else {
		cmd = exec.Command("ffplay", path, "-autoexit", "-nodisp", "-hide_banner", "-loglevel", "warning")
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start music: %v", err)
		return
	}

	musicPid = cmd.Process
	log.Println("Playing elevator", path)
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
