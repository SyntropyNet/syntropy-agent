package agent

import (
	"encoding/json"
	"time"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
)

func (a *Agent) addCommand(cmd common.Command) error {
	a.commands[cmd.Name()] = cmd
	return nil
}

func (a *Agent) processCommand(raw []byte) {
	var req common.MessageHeader
	if err := json.Unmarshal(raw, &req); err != nil {
		logger.Error().Println(pkgName, "json message unmarshal error: ", err)
		return
	}

	cmd, ok := a.commands[req.MsgType]
	if !ok {
		logger.Warning().Printf("%s Command '%s' not found\n", pkgName, req.MsgType)
		logger.Message().Println(pkgName, "Received:", string(raw))
		return
	}

	logger.Message().Println(pkgName, "Received: ", string(raw))
	started := time.Now()
	err := cmd.Exec(raw)
	if err != nil {
		logger.Error().Printf("%s Command '%s' failed: %s\n", pkgName, req.MsgType, err.Error())
	}
	logger.Info().Printf("%s Command '%s' completed in %s.", pkgName, req.MsgType, time.Now().Sub(started))
}
