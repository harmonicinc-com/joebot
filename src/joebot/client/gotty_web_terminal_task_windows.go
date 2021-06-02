package client

import (
	"net"

	"github.com/harmonicinc-com/joebot/task"
	"github.com/pkg/errors"
)

type GottyWebTerminalTask struct {
	handleClient *Client
	*task.Task
}

func NewGottyWebTerminalTask(client *Client) *GottyWebTerminalTask {
	return &GottyWebTerminalTask{
		client,
		task.NewTask(client.ctx, task.GottyWebTerminalRequest, client.logger),
	}
}

func (t *GottyWebTerminalTask) Handle(body []byte, stream net.Conn) error {
	return errors.New("The client OS is Windows which does not support gotty")
}
