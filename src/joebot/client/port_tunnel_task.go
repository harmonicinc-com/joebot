package client

import (
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/ginuerzh/gost"
	"github.com/harmonicinc-com/joebot/models"
	"github.com/harmonicinc-com/joebot/task"
	"github.com/harmonicinc-com/joebot/utils"
	"github.com/pkg/errors"
)

type PortTunnelTask struct {
	handleClient *Client
	*task.Task
}

func NewPortTunnelTask(client *Client) *PortTunnelTask {
	return &PortTunnelTask{
		client,
		task.NewTask(client.ctx, task.PortTunnelRequest, client.logger),
	}
}

func (t *PortTunnelTask) Handle(body []byte, stream net.Conn) error {
	if t.handleClient == nil {
		return errors.New("handleClient param not set")
	}

	var tunnel models.PortTunnelInfo
	err := utils.BytesToStruct(body, &tunnel)
	if err != nil {
		return errors.Wrap(err, "Unable to decode request body into PortTunnelInfo object")
	}

	chain := gost.NewChain(
		gost.Node{
			Protocol:  "forward",
			Transport: "ssh",
			Addr:      t.handleClient.serverIP + ":" + strconv.Itoa(tunnel.GostServerPort),
			Client: &gost.Client{
				Connector:   gost.SSHRemoteForwardConnector(),
				Transporter: gost.SSHForwardTransporter(),
			},
		},
	)
	ln, err := gost.TCPRemoteForwardListener(":"+strconv.Itoa(tunnel.ServerPort), chain)
	if err != nil {
		log.Fatal(err)
	}

	s := &gost.Server{Listener: ln}
	h := gost.TCPRemoteForwardHandler(
		"localhost:" + strconv.Itoa(tunnel.ClientPort),
	)
	h.Init(
		gost.AddrHandlerOption(ln.Addr().String()),
		gost.ChainHandlerOption(chain),
	)
	go func() {
		err := s.Serve(h)
		if err != nil {
			fmt.Println(err)
		}
		t.handleClient.RemoveTunnel(s)
	}()
	t.handleClient.AddTunnel(s)

	return task.ConfirmTaskComplete(stream)
}
