package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/SyntropyNet/syntropy-agent-go/agent"
	"github.com/SyntropyNet/syntropy-agent-go/config"
	"github.com/SyntropyNet/syntropy-agent-go/logger"
)

const fullAppName = "Syntropy Stack Agent"

func main() {
	execName := os.Args[0]

	showVersionAndExit := flag.Bool("version", false, "Show version and exit")
	flag.Parse()
	if *showVersionAndExit {
		fmt.Printf("%s (%s):\t%s\n\n", fullAppName, execName, config.GetFullVersion())
		return
	}

	config.Init()
	defer config.Close()

	// TODO: init Wireguard (see pyroyte2.Wireguard())

	syntropyNetAgent, err := agent.NewAgent()
	if err != nil {
		log.Fatal("Could not create ", fullAppName, err)
	}

	// Loggers started with ERR level to stderr
	// After creating agent instance reconfigure loggers

	// NotYet: do not spam controller in development stage
	// NotYet: logger.SetControllerWriter(syntropyNetAgent)
	logger.Setup(logger.DebugLevel, os.Stdout)
	logger.Info().Println(fullAppName, execName, config.GetFullVersion(), "started")

	//Start main agent loop (forks to goroutines internally)
	syntropyNetAgent.Loop()

	// Wait for SIGINT or SIGKILL to terminate app
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	logger.Info().Println(fullAppName, " terminating")

	// Stop and cleanup
	syntropyNetAgent.Stop()
}
