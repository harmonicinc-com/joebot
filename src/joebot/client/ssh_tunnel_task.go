package client

import (
	"net"
	"strconv"
	"time"

	"github.com/harmonicinc-com/joebot/models"
	"github.com/harmonicinc-com/joebot/utils"
	"github.com/pkg/errors"

	"github.com/harmonicinc-com/joebot/task"
)

type SSHTunnelTask struct {
	*task.Task
}

func NewSSHTunnelTask(client *Client) *SSHTunnelTask {
	return &SSHTunnelTask{
		task.NewTask(client.ctx, task.SSHTunnelRequest, client.logger),
	}
}

func (t *SSHTunnelTask) Handle(body []byte, stream net.Conn) error {
	var sshTunnel models.PortTunnelInfo
	sshTunnel.ClientPort = 22

	if !utils.IsPortOccupied(sshTunnel.ClientPort) {
		return errors.New("No SSH Service Listening On Port: " + strconv.Itoa(sshTunnel.ClientPort))
	}
	return task.SendObject(utils.StructToBytes(sshTunnel), stream, 10*time.Second)
}
