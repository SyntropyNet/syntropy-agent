package config

import "time"

const (
	ControllerSaas = iota
	ControllerScript
	ControllerBlockchain
	ControllerUnknown
)

const (
	ContainerTypeDocker     = "docker"
	ContainerTypeKubernetes = "kubernetes"
	ContainerTypeHost       = "host"
)

func GetControllerType() int {
	return cache.controllerType
}

func GetControllerName(ctype int) string {
	switch ctype {
	case ControllerSaas:
		return "SaaS (cloud)"
	case ControllerScript:
		return "Script"
	case ControllerBlockchain:
		return "Blockchain"
	default:
		return "Unknown"
	}
}

func GetDebugLevel() int {
	return cache.debugLevel
}

func GetAgentToken() string {
	return cache.apiKey
}

func GetCloudURL() string {
	return cache.cloudURL
}

func GetAgentName() string {
	return cache.agentName
}

func GetAgentProvider() int {
	return cache.agentProvider
}

func GetAgentCategory() string {
	return cache.agentCategory
}

func GetServicesStatus() bool {
	return cache.servicesStatus
}

func GetAgentTags() []string {
	if len(cache.agentTags) > 0 {
		return cache.agentTags
	} else {
		return []string{}
	}
}

func GetNetworkIDs() []string {
	if len(cache.networkIDs) > 0 {
		return cache.networkIDs
	} else {
		return []string{}
	}
}

func GetPublicIp() string {
	if time.Now().After(cache.publicIP.updated.Add(ipUpdatePeriod)) {
		updatePublicIp()
	}

	return cache.publicIP.ip
}

func GetPortsRange() (uint16, uint16) {
	return cache.portsRange.start, cache.portsRange.end
}

func GetDeviceID() string {
	return cache.deviceID
}

func GetContainerType() string {
	return cache.containerType
}

func GetLocationLatitude() float32 {
	return cache.location.Latitude
}

func GetLocationLongitude() float32 {
	return cache.location.Longitude
}

func CleanupOnExit() bool {
	return cache.cleanupOnExit
}

func GetHostAllowedIPs() []AllowedIPEntry {
	return cache.allowedIPs
}
