package musicplayer

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

var MusicPid *os.Process
var dingPid *os.Process

func PlayMusic(path string, pid *os.Process, loop_music bool) {
	if path == "" {
		if pid != nil {
			pid.Signal(syscall.SIGCONT)
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

	pid = cmd.Process
	log.Println("Playing elevator", path)
}

func PauseMusic(pid *os.Process) {
	if pid != nil {
		pid.Signal(syscall.SIGSTOP)
		log.Println("Elevator music paused")
	}
}

func ResumeMusic(pid *os.Process) {
	if pid != nil {
		pid.Signal(syscall.SIGCONT)
		log.Println("Elevator music resumed")
	}
}

func StopMusic(pid *os.Process) {
	if pid != nil {
		pid.Signal(syscall.SIGINT)
		log.Println("Elevator music stopped")
	}
}

func PlayDing() {
	PlayMusic("media/ding.opus", dingPid, false)
}
