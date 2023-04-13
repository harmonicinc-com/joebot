package client

import (
	"context"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ginuerzh/gost"

	"github.com/harmonicinc-com/joebot/handler"
	"github.com/harmonicinc-com/joebot/models"
	"github.com/harmonicinc-com/joebot/task"
	"github.com/harmonicinc-com/joebot/utils"
	"github.com/hashicorp/yamux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Client struct {
	logger *logrus.Logger

	Tags []string

	conn    net.Conn
	session *yamux.Session

	serverIP          string
	serverPort        int
	reconnectInterval time.Duration

	allowedPortRangeLBound int
	allowedPortRangeUBound int
	portsManager           *utils.PortsManager

	gostTunnels    []*gost.Server
	tunnelListLock chan bool

	novncWebsocketServer *http.Server

	FilebrowserDefaultDir string
	filebrowserServer     *http.Server

	ctx  context.Context
	stop context.CancelFunc
}

func NewClient(serverIP string, serverPort int, allowedPortRangeLBound int, allowedPortRangeUBound int, tags []string, logger *logrus.Logger) *Client {
	if logger == nil {
		logger = logrus.New()
	}

	client := &Client{}
	client.Tags = tags
	client.logger = logger
	client.serverIP = serverIP
	client.serverPort = serverPort
	client.reconnectInterval = 5 * time.Second

	client.allowedPortRangeLBound = allowedPortRangeLBound
	client.allowedPortRangeUBound = allowedPortRangeUBound
	client.portsManager = utils.NewPortsManager()
	client.SetAllowedPortRange(allowedPortRangeLBound, allowedPortRangeUBound)

	client.tunnelListLock = make(chan bool, 1)
	client.tunnelListLock <- true

	client.ctx, client.stop = context.WithCancel(context.Background())

	return client
}

func (client *Client) SetAllowedPortRange(lbound int, ubound int) {
	if lbound < 0 || ubound < 0 || lbound > ubound {
		panic("Invalid Allowed Port Range")
	}
	if lbound == 0 && ubound == 65535 {
		return
	}
	for i := lbound; i <= ubound; i++ {
		client.portsManager.AddAllowedPort(i)
	}
}

func (client *Client) AddTunnel(tunnel *gost.Server) {
	<-client.tunnelListLock
	defer func() { client.tunnelListLock <- true }()

	client.gostTunnels = append(client.gostTunnels, tunnel)
}

func (client *Client) RemoveTunnel(tunnel *gost.Server) {
	<-client.tunnelListLock
	defer func() { client.tunnelListLock <- true }()

	for i, t := range client.gostTunnels {
		if t.Addr().String() == tunnel.Addr().String() {
			client.gostTunnels = append(client.gostTunnels[:i], client.gostTunnels[i+1:]...)
		}
	}
}

func (client *Client) ExitIfError(err error, message string) bool {
	if err != nil {
		if message != "" {
			err = errors.Wrap(err, message)
		}
		client.logger.Error(err)
		client.Stop()

		client.Reconnect()

		return true
	}

	return false
}

func (client *Client) Reconnect() {
	go func(client *Client) {
		client.logger.Info("Sleep Before Reconnecting")
		time.Sleep(client.reconnectInterval)
		client.logger.Info("Reconnecting...")
		c := NewClient(client.serverIP, client.serverPort, client.allowedPortRangeLBound, client.allowedPortRangeUBound, client.Tags, client.logger)
		c.FilebrowserDefaultDir = client.FilebrowserDefaultDir
		c.Start()
	}(client)
}

func (client *Client) Start() {
	var err error

	// Get a TCP connection
	client.conn, err = net.Dial("tcp", client.serverIP+":"+strconv.Itoa(client.serverPort))
	if client.ExitIfError(err, "Unable to connect to server: "+client.serverIP+":"+strconv.Itoa(client.serverPort)) {
		return
	}

	client.logger.Info("Client Dialed.")
	// Setup client side of yamux
	client.session, err = yamux.Client(client.conn, nil)
	if client.ExitIfError(err, "Unable to create yamux session") {
		return
	}

	inHandler := handler.NewIncomingRequestHandler(client.ctx, client.session, client.logger)
	inHandler.OnError(func(err error) {
		client.ExitIfError(err, "Handler OnError")
	})
	inHandler.RegisterTask(NewPortTunnelTask(client))
	inHandler.RegisterTask(NewSSHTunnelTask(client))
	inHandler.RegisterTask(NewNovncTask(client))
	inHandler.RegisterTask(NewGottyWebTerminalTask(client))
	inHandler.RegisterTask(NewFilebrowserTask(client))
	inHandler.Start()

	client.UpdateClientInfo()
}

func (client *Client) Stop() {
	<-client.tunnelListLock
	defer func() { client.tunnelListLock <- true }()

	for _, t := range client.gostTunnels {
		t.Close()
	}
	client.gostTunnels = []*gost.Server{}

	if client.novncWebsocketServer != nil {
		client.novncWebsocketServer.Shutdown(context.Background())
		client.novncWebsocketServer = nil
	}

	if client.filebrowserServer != nil {
		client.filebrowserServer.Shutdown(context.Background())
		client.filebrowserServer = nil
	}

	client.stop()
}

func (client *Client) UpdateClientInfo() {
	clientInfo := models.ClientInfo{}
	clientInfo.IP = strings.Split(client.conn.LocalAddr().String(), ":")[0]
	clientInfo.HostName, _ = os.Hostname()
	clientInfo.Username = os.Getenv("USER")
	clientInfo.Tags = client.Tags
	_, err := task.NewTask(client.ctx, task.ClientInfoUpdateRequest, client.logger).Request(client.session, utils.StructToBytes(clientInfo))
	if client.ExitIfError(err, "Failed To Update Client Info") {
		return
	}
}
