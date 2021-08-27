package saas

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/SyntropyNet/syntropy-agent-go/internal/config"
	"github.com/SyntropyNet/syntropy-agent-go/internal/logger"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/common"
	"github.com/SyntropyNet/syntropy-agent-go/pkg/state"
	"github.com/gorilla/websocket"
)

const pkgName = "Saas Controller. "
const (
	stopped = iota
	connecting
	running
)

type CloudController struct {
	sync.Mutex // this lock makes Write thread safe
	state.StateMachine
	ws      *websocket.Conn
	url     string
	token   string
	version string
}

// NewController allocates instance of Software-As-A-Service
// (aka WSS) controller
func NewController() (common.Controller, error) {
	// Note: config package returns already validated values and no need to validate them here
	cc := CloudController{
		url:     config.GetCloudURL(),
		token:   config.GetAgentToken(),
		version: config.GetVersion(),
	}
	cc.SetState(stopped)

	err := cc.connect()
	if err != nil {
		return nil, err
	}

	return &cc, nil
}

func (cc *CloudController) connect() (err error) {
	cc.SetState(connecting)
	url := url.URL{Scheme: "wss", Host: cc.url, Path: "/"}
	headers := http.Header(make(map[string][]string))

	// Without these headers connection will be ignored silently
	headers.Set("authorization", cc.token)
	headers.Set("x-deviceid", config.GetDeviceID())
	headers.Set("x-deviceip", config.GetPublicIp())
	headers.Set("x-devicename", config.GetAgentName())
	headers.Set("x-devicestatus", "OK")
	headers.Set("x-agenttype", "Linux")
	headers.Set("x-agentversion", cc.version)

	for {
		var resp *http.Response
		var httpCode int
		cc.ws, resp, err = websocket.DefaultDialer.Dial(url.String(), headers)
		if err != nil {
			if resp != nil {
				httpCode = resp.StatusCode
			}
			logger.Error().Printf("%s ConnectionError: %s (HTTP: %d)\n", pkgName, err.Error(), httpCode)
			// Add some randomised sleep, so if controller was down
			// the reconnecting agents could DDOS the controller
			delay := time.Duration(rand.Int31n(10000)) * time.Millisecond
			logger.Warning().Println(pkgName, "Reconnecting in ", delay)
			time.Sleep(delay)
			continue
		}

		cc.SetState(running)
		break
	}

	return nil
}

func (cc *CloudController) Recv() ([]byte, error) {
	if cc.GetState() == stopped {
		return nil, fmt.Errorf("controller is not running")
	}

	// In this application we have only one reader, so no need to lock here

	for {
		msgtype, msg, err := cc.ws.ReadMessage()

		switch {
		case err == nil:
			// successfully received message
			if msgtype != websocket.TextMessage {
				logger.Warning().Println(pkgName, "Received unexpected message type ", msgtype)
			}
			logger.Debug().Println(pkgName, "Received: ", string(msg))
			return msg, nil

		case cc.GetState() == stopped:
			// The connection is closed - simulate EOF
			logger.Debug().Println(pkgName, "EOF")
			return nil, io.EOF
		}

		// reconnect and continue receiving
		// NOTE: connect is blocking and will block untill a connection is established
		cc.connect()
	}
}

func (cc *CloudController) Write(b []byte) (n int, err error) {
	if controllerState := cc.GetState(); controllerState != running {
		logger.Warning().Println(pkgName, "Controller is not running. Current state: ", controllerState)
		return 0, fmt.Errorf("controller is not running (%d)", controllerState)
	}

	/*
		gorilla/websocket concurency:
			Connections support one concurrent reader and one concurrent writer.
			Applications are responsible for ensuring that no more than one goroutine calls the write methods
	*/
	cc.Lock()
	defer cc.Unlock()

	logger.Debug().Println(pkgName, "Sending: ", string(b))
	err = cc.ws.WriteMessage(websocket.TextMessage, b)
	if err != nil {
		logger.Error().Println(pkgName, "Send error: ", err)
	} else {
		n = len(b)
	}
	return n, err
}

// Close closes websocket connection to saas backend
func (cc *CloudController) Close() error {
	if cc.GetState() == stopped {
		// cannot close already closed connection
		return fmt.Errorf("controller already closed")
	}
	cc.SetState(stopped)

	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	err := cc.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		logger.Error().Println(pkgName, "connection close error: ", err)
	}

	cc.ws.Close()
	return nil
}
