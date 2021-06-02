package server

import (
	"context"
	"net"
	"time"

	"github.com/harmonicinc-com/joebot/utils"

	"github.com/harmonicinc-com/joebot/handler"
	"github.com/harmonicinc-com/joebot/models"
	"github.com/harmonicinc-com/joebot/task"
	"github.com/hashicorp/yamux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Client struct {
	logger *logrus.Logger

	ID      string
	conn    *net.Conn
	server  *Server
	session *yamux.Session
	streams []*yamux.Stream

	ctx  context.Context
	stop context.CancelFunc

	Info models.ClientInfo
}

func NewClient(id string, server *Server, conn *net.Conn, logger *logrus.Logger) *Client {
	if logger == nil {
		logger = logrus.New()
	}

	client := &Client{}
	client.ID = id
	client.logger = logger
	client.conn = conn
	client.server = server
	client.ctx, client.stop = context.WithCancel(context.Background())

	client.Info = models.ClientInfo{}
	client.Info.ID = id
	client.Info.Tags = []string{}
	client.Info.PortTunnels = []models.PortTunnelInfo{}

	logger.Info("Init new client")
	return client
}

func (client *Client) UpdateInfo(info models.ClientInfo) {
	if info.Tags != nil {
		client.Info.Tags = info.Tags
	}
	client.Info.IP = info.IP
	client.Info.HostName = info.HostName
	client.Info.Username = info.Username
}

func (client *Client) ExitIfError(err error, message string) bool {
	if err != nil {
		if message != "" {
			err = errors.Wrap(err, message)
		}

		client.logger.WithField("Client ID", client.ID).Error(err)
		client.Stop()
		client.server.RemoveClient(client.ID)

		return true
	}

	return false
}

func (client *Client) Start() {
	// Setup server side of yamux
	session, err := yamux.Server(*(client.conn), nil)
	if client.ExitIfError(err, "Unable to create yamux session") {
		return
	}
	client.session = session

	inHandler := handler.NewIncomingRequestHandler(client.ctx, client.session, client.logger)
	inHandler.OnError(func(err error) {
		client.ExitIfError(err, "")
	})
	inHandler.RegisterTask(NewClientInfoUpdateTask(client))
	inHandler.Start()

	if _, err := client.CreateSSHTunnel(); err != nil {
		client.logger.Info(errors.Wrap(err, "Failed To Create SSH Tunnel"))
	}
	if _, err = client.CreateGottyWebTerminal(); err != nil {
		client.logger.Info(errors.Wrap(err, "Failed To Create Gotty Web Terminal Tunnel"))
	}
	if _, err = client.CreateNovncWebsocketTunnel(5901); err != nil {
		client.logger.Info(errors.Wrap(err, "Failed To Create NoVNC Websocket Tunnel"))
	}
	if _, err = client.CreateFilebrowser(); err != nil {
		client.logger.Info(errors.Wrap(err, "Failed To Create Web filebrowser"))
	}
}

func (client *Client) CreateSSHTunnel() (models.PortTunnelInfo, error) {
	var err error
	var sshTunnel models.PortTunnelInfo
	if client.Info.SSHTunnel != nil {
		return sshTunnel, errors.New("Failed to create SSH tunnel | tunnel already exists")
	}

	client.logger.WithField("Client ID", client.ID).Info("Creating SSH Tunnel")
	stream, err := task.NewTask(client.ctx, task.SSHTunnelRequest, client.logger).Request(client.session, []byte{})
	if err != nil {
		return sshTunnel, errors.Wrap(err, "SSH Tunnel Request Failed")
	}
	body, err := task.ReceiveObject(stream, 10*time.Second)
	if err != nil {
		return sshTunnel, errors.Wrap(err, "Failed To Locate Client's SSH")
	}
	err = utils.BytesToStruct(body, &sshTunnel)
	if err != nil {
		return sshTunnel, err
	}

	portTunnelInfo, err := client.CreateTunnel(sshTunnel.ClientPort)
	if err != nil {
		return sshTunnel, errors.Wrap(err, "Failed To Create SSH Tunnel")
	}

	client.Info.SSHTunnel = &portTunnelInfo
	return sshTunnel, err
}

func (client *Client) CreateFilebrowser() (models.FilebrowserInfo, error) {
	var err error
	var fbInfo models.FilebrowserInfo
	if client.Info.FilebrowserInfo != nil {
		return fbInfo, errors.New("Failed to create web filebrowser | service already exists")
	}

	client.logger.WithField("Client ID", client.ID).Info("Creating web filebrowser")
	stream, err := task.NewTask(client.ctx, task.FilebrowserRequest, client.logger).Request(client.session, []byte{})
	if err != nil {
		return fbInfo, err
	}

	body, err := task.ReceiveObject(stream, time.Second*10)
	if err != nil {
		return fbInfo, errors.Wrap(err, "Failed To Receive Web filebrowser From Client")
	}
	if err = utils.BytesToStruct(body, &fbInfo); err != nil {
		return fbInfo, err
	}

	portTunnelInfo, err := client.CreateTunnel(fbInfo.FilebrowserPort)
	if err != nil {
		return fbInfo, errors.Wrap(err, "Failed To Create Tunnel To Web filebrowser")
	}
	fbInfo.PortTunnelOnHost = portTunnelInfo

	client.Info.FilebrowserInfo = &fbInfo
	return fbInfo, nil
}

func (client *Client) CreateGottyWebTerminal() (models.GottyWebTerminalInfo, error) {
	var err error
	var wtInfo models.GottyWebTerminalInfo
	if client.Info.GottyWebTerminalInfo != nil {
		return wtInfo, errors.New("Failed to create Gotty web terminal tunnel | service already exists")
	}

	client.logger.WithField("Client ID", client.ID).Info("Creating gotty Web Terminal")
	stream, err := task.NewTask(client.ctx, task.GottyWebTerminalRequest, client.logger).Request(client.session, []byte{})
	if err != nil {
		return wtInfo, err
	}

	body, err := task.ReceiveObject(stream, time.Second*10)
	if err != nil {
		return wtInfo, errors.Wrap(err, "Failed To Receive Web Terminal Info From Client")
	}
	if err = utils.BytesToStruct(body, &wtInfo); err != nil {
		return wtInfo, err
	}

	portTunnelInfo, err := client.CreateTunnel(wtInfo.GottyWebTerminalPort)
	if err != nil {
		return wtInfo, errors.Wrap(err, "Failed To Create Tunnel To Gotty Web Terminal")
	}
	wtInfo.PortTunnelOnHost = portTunnelInfo

	client.Info.GottyWebTerminalInfo = &wtInfo
	return wtInfo, nil
}

func (client *Client) CreateNovncWebsocketTunnel(clientVncPort int) (models.NovncWebsocketInfo, error) {
	var err error
	var novncWebsocketInfo models.NovncWebsocketInfo
	if client.Info.NovncWebsocketInfo != nil {
		return novncWebsocketInfo, errors.New("Failed to create novnc tunnel | service already exists")
	}

	client.logger.WithField("Client ID", client.ID).Infof("Creating novnc Websocket Tunnel | Client VNC Port %d", clientVncPort)

	novncWebsocketInfo.VncServerPort = clientVncPort
	stream, err := task.NewTask(client.ctx, task.NovncRequest, client.logger).Request(client.session, utils.StructToBytes(novncWebsocketInfo))
	if err != nil {
		return novncWebsocketInfo, err
	}

	body, err := task.ReceiveObject(stream, time.Second*10)
	if err != nil {
		return novncWebsocketInfo, err
	}
	err = utils.BytesToStruct(body, &novncWebsocketInfo)
	if err != nil {
		return novncWebsocketInfo, err
	}

	portTunnelInfo, err := client.CreateTunnel(novncWebsocketInfo.NovncWebsocketPort)
	if err != nil {
		return novncWebsocketInfo, errors.Wrap(err, "Failed To Create Tunnel To NoVNC Websocket")
	}
	novncWebsocketInfo.PortTunnelOnHost = portTunnelInfo

	client.Info.NovncWebsocketInfo = &novncWebsocketInfo
	return novncWebsocketInfo, nil
}

func (client *Client) CreateTunnel(clientPort int) (models.PortTunnelInfo, error) {
	// Avoid overloading the gost tunnel service by ensure there is at most one client to create tunnel
	gostTunnelService := client.server.GetTunnelService()
	gostTunnelService.Lock()
	defer gostTunnelService.UnLock()

	var err error
	var tunnel models.PortTunnelInfo

	client.logger.WithField("Client ID", client.ID).Infof("Creating Tunnel To Client Port %d", clientPort)
	//Check if the client port is already tunnelled
	for _, t := range client.Info.PortTunnels {
		if t.ClientPort == clientPort {
			return t, nil
		}
	}

	tunnel.GostServerPort = gostTunnelService.Port
	tunnel.ServerPort, err = client.server.portsManager.ReservePort()
	if err != nil {
		return tunnel, err
	}
	tunnel.ClientPort = clientPort

	stream, err := task.NewTask(client.ctx, task.PortTunnelRequest, client.logger).Request(client.session, utils.StructToBytes(tunnel))
	if err != nil {
		client.server.portsManager.ReleasePort(tunnel.ServerPort)
		return tunnel, errors.Wrap(err, "Failed To Instruct Client To Do Port Forwarding")
	}

	err = task.WaitTaskCompleteSignal(60*time.Second, stream)
	if err != nil {
		client.server.portsManager.ReleasePort(tunnel.ServerPort)
		return tunnel, errors.New("Client Failed To Create Tunnel | Client ID: " + client.ID)
	}
	client.logger.WithField("Client ID", client.ID).Infof("Created Tunnel | Host Port: %d | Client Port: %d", tunnel.ServerPort, tunnel.ClientPort)
	client.Info.PortTunnels = append(client.Info.PortTunnels, tunnel)
	return tunnel, nil
}

func (client *Client) Stop() error {
	defer func() {
		for _, t := range client.Info.PortTunnels {
			client.server.portsManager.ReleasePort(t.ServerPort)
		}
		client.Info.PortTunnels = []models.PortTunnelInfo{}
	}()

	client.stop()
	if client.session.IsClosed() {
		return nil
	}

	client.session.GoAway()
	for _, stream := range client.streams {
		err := stream.Close()
		if err != nil {
			err = errors.Wrap(err, "Unable to stop client's stream")
			client.logger.Error(err)
		}
	}
	client.session.Close()
	return (*client.conn).Close()
}
