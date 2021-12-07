package config

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

func GetOwnerAddress() string {
	return cache.ownerAddress
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

func GetPortsRange() (uint16, uint16) {
	return cache.portsRange.start, cache.portsRange.end
}

func GetInterfaceMTU() uint32 {
	return cache.mtu
}

func CreateIptablesRules() bool {
	return cache.createIptablesRules
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

func IsVPNClient() bool {
	return cache.vpnClient
}

func SetRerouteThresholds(diff, ratio float32) {
	cache.rerouteThresholds.diff = diff
	cache.rerouteThresholds.ratio = ratio
}

func RerouteThresholds() (float32, float32) {
	return cache.rerouteThresholds.diff, cache.rerouteThresholds.ratio
}
