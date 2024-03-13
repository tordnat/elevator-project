package main

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

func PlayElevatorMusic(musicPath string, signalChannel chan os.Signal) {
	var musicPid *os.Process
	cmd := exec.Command("ffplay", musicPath, "-autoexit", "-nodisp", "-hide_banner", "-loglevel", "warning", "-loop", "0")

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start music: %v", err)
		return
	}
	log.Println("Playing music from: ", musicPath)
	musicPid = cmd.Process

	for {
		sig := <-signalChannel
		switch sig {
		case syscall.SIGINT: // Stop
			musicPid.Signal(sig)
			return
		case syscall.SIGSTOP: // Pause
			musicPid.Signal(sig)
		case syscall.SIGCONT: // Resume
			musicPid.Signal(syscall.SIGCONT)

		}
	}
}

func PlayFloorArrivalDing(musicPath string) {
	cmd := exec.Command("ffplay", musicPath, "-autoexit", "-nodisp", "-hide_banner", "-loglevel", "warning")
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start music: %v", err)
		return
	}
}
