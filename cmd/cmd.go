package cmd

import (
	"flag"
	"fmt"
)

func InitCommandLineArgs(args []string) (int, string) {
	var port int
	flag.IntVar(&port, "port", 15657, "Specifies the port to connect to elevator on, default is 15657")
	var id int
	flag.IntVar(&id, "id", 0, "Specifies the unique id of elevator, default is 0")
	flag.Parse()

	fmt.Println("Port for connecting to elevator is", port, " ID of elevator is ", id)

	return port, fmt.Sprint(id)
}
