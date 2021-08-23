package getinfo

import (
	"encoding/json"
	"io"

	"github.com/SyntropyNet/syntropy-agent-go/config"
	"github.com/SyntropyNet/syntropy-agent-go/controller"
	"github.com/SyntropyNet/syntropy-agent-go/docker"
)

const cmd = "GET_INFO"

type getInfoRequest struct {
	controller.MessageHeader
	Data interface{} `json:"data,omitempty"`
}

type getInfoResponce struct {
	controller.MessageHeader
	Data struct {
		Provider          int      `json:"agent_provider,omitempty"` // 0 is not used and do not send
		Status            bool     `json:"service_status"`
		Tags              []string `json:"agent_tags"`
		ExternalIP        string   `json:"external_ip"`
		LocationLatitude  string   `json:"location_lat"`
		LocationLongitude string   `json:"location_lon"`

		NetworkInfo   []docker.DockerNetworkInfoEntry   `json:"network_info"`
		ContainerInfo []docker.DockerContainerInfoEntry `json:"container_info"`
	} `json:"data"`
}

type getInfo struct {
	w io.Writer
}

func New(w io.Writer) controller.Command {
	return &getInfo{
		w: w,
	}
}

func (obj *getInfo) Name() string {
	return cmd
}

func (obj *getInfo) Exec(raw []byte) error {
	var req getInfoRequest
	err := json.Unmarshal(raw, &req)
	if err != nil {
		return err
	}

	resp := getInfoResponce{
		MessageHeader: req.MessageHeader,
	}

	resp.Data.Provider = config.GetAgentProvider()
	resp.Data.Status = config.GetServicesStatus()
	resp.Data.Tags = config.GetAgentTags()
	resp.Data.ExternalIP = config.GetPublicIp()
	resp.Data.LocationLatitude = config.GetLocationLatitude()
	resp.Data.LocationLongitude = config.GetLocationLongitude()
	resp.Data.NetworkInfo = docker.NetworkInfo()
	resp.Data.ContainerInfo = docker.ContainerInfo()

	arr, err := json.Marshal(&resp)
	if err != nil {
		return err
	}

	obj.w.Write(arr)

	return err
}
