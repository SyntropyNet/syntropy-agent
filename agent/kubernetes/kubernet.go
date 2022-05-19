package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/SyntropyNet/syntropy-agent/agent/common"
	"github.com/SyntropyNet/syntropy-agent/internal/config"
	"github.com/SyntropyNet/syntropy-agent/internal/env"
	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/build/kubernetes"
)

// TODO (later): in future think about optimising binary size
// and using GO stdlib kubernetes package
// (premature optimisation is the root of all evil)
const (
	pkgName = "Kubernetes. "
	cmd     = "KUBERNETES_SERVICE_INFO"
)

type kubernet struct {
	writer io.Writer
	klient *kubernetes.Client
	msg    kubernetesInfoMessage
	ctx    context.Context
}

func New(w io.Writer) common.Service {
	kub := kubernet{
		writer: w,
	}
	kub.msg.MsgType = cmd
	kub.msg.ID = env.MessageDefaultID

	return &kub
}

func (obj *kubernet) Name() string {
	return cmd
}

func (obj *kubernet) execute() {
	services := obj.monitorServices()
	if !cmp.Equal(services, obj.msg.Data) {
		obj.msg.Data = services
		obj.msg.Now()
		raw, err := json.Marshal(obj.msg)
		if err != nil {
			logger.Error().Println(pkgName, "json marshal", err)
			return
		}
		logger.Debug().Println(pkgName, "Sending: ", string(raw))
		obj.writer.Write(raw)
	}
}

func (obj *kubernet) Run(ctx context.Context) error {
	if obj.klient != nil {
		return fmt.Errorf("kubernetes watcher already running")
	}
	obj.ctx = ctx

	err := obj.initClient()
	if err != nil {
		logger.Error().Println(pkgName, err)
		return err
	}

	go func() {
		ticker := time.NewTicker(config.PeerCheckTime())
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Debug().Println(pkgName, "stopping", cmd)
				obj.klient.Close()
				obj.klient = nil
				return
			case <-ticker.C:
				obj.execute()
			}
		}
	}()
	return nil
}
