package v1

const (
	keyError   = "error"
	keyMessage = "message"
	keyTotal   = "total"
	keyCount   = "count"
	keyNodes   = "nodes"
	keyRunning = "running"
)

const (
	msgTryAgainLater            = "Try again later"
	msgJSONInvalid              = "JSON data is invalid"
	msgInvalidLimit             = "Invalid limit"
	msgInvalidPage              = "Invalid page"
	msgUserDoesNotExist         = "User does not exist"
	msgKeyInvalid               = "Key is invalid"
	msgServerAddressInvalid     = "Server address is invalid"
	msgServerPortInvalid        = "Server port is invalid"
	msgErrAddingWireguardPeer   = "Error adding wireguard peer"
	msgErrGeneratingOLSRDConfig = "Error generating olsrd config"
	msgErrGettingTunnel         = "Error getting tunnel"
	msgErrGettingTunnelCount    = "Error getting tunnel count"
	msgErrReloadingDNS          = "Error reloading DNS"
	msgErrReloadingOLSRD        = "Error reloading olsrd"
)
