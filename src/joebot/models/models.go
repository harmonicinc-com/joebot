package models

type ClientInfo struct {
	ID                   string                `json:"id"`
	IP                   string                `json:"ip"`
	HostName             string                `json:"host_name"`
	Tags                 []string              `json:"tags"`
	Username             string                `json:"username"`
	PortTunnels          []PortTunnelInfo      `json:"port_tunnels"`
	SSHTunnel            *PortTunnelInfo       `json:"ssh_tunnel,omitempty"`
	NovncWebsocketInfo   *NovncWebsocketInfo   `json:"novnc_websocket_info,omitempty"`
	GottyWebTerminalInfo *GottyWebTerminalInfo `json:"gotty_web_terminal_info,omitempty"`
	FilebrowserInfo      *FilebrowserInfo      `json:"filebrowser_info,omitempty"`
}

type ClientCollection struct {
	Clients []ClientInfo `json:"clients"`
}

type PortTunnelInfo struct {
	GostServerPort int `json:"gost_server_port"`
	ServerPort     int `json:"server_port"`
	ClientPort     int `json:"client_port"`
}

type NovncWebsocketInfo struct {
	VncServerPort      int            `json:"vnc_server_port"`
	NovncWebsocketPort int            `json:"novnc_websocket_port"`
	PortTunnelOnHost   PortTunnelInfo `json:"port_tunnel"`
}

type GottyWebTerminalInfo struct {
	GottyWebTerminalPort int            `json:"gotty_web_terminal_port"`
	PortTunnelOnHost     PortTunnelInfo `json:"port_tunnel"`
}

type FilebrowserInfo struct {
	DefaultDirectory string         `json:"default_directory"`
	FilebrowserPort  int            `json:"filebrowser_port"`
	PortTunnelOnHost PortTunnelInfo `json:"port_tunnel"`
}

type Address struct {
	IP   string `json:"IP"`
	Port int    `json:"Port"`
}

type BulkInstallInfo struct {
	JoebotServerIP   string    `json:"JoebotServerIP"`
	JoebotServerPort int       `json:"JoebotServerPort"`
	Addresses        []Address `json:"Addresses"`
	Username         string    `json:"Username"`
	Password         string    `json:"Password"`
	Key              string    `json:"Key"`
}
