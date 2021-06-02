// +build !windows

package client

import (
	"net"
	"os"
	"strconv"
	"time"

	"github.com/harmonicinc-com/joebot/models"
	"github.com/harmonicinc-com/joebot/task"
	"github.com/harmonicinc-com/joebot/utils"
	"github.com/pkg/errors"

	gotty_localcommand "github.com/yudai/gotty/backend/localcommand"
	gotty_server "github.com/yudai/gotty/server"
	gotty_utils "github.com/yudai/gotty/utils"
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

//TBD: https://stackoverflow.com/questions/39508086/golang-exec-background-process-and-get-its-pid

func (t *GottyWebTerminalTask) Handle(body []byte, stream net.Conn) error {
	if t.handleClient == nil {
		return errors.New("handleClient param not set")
	}

	freePort, err := t.handleClient.portsManager.ReservePort()
	if err != nil {
		return errors.Wrap(err, "Unable To Find Free Port For Gotty Service")
	}

	appOptions := &gotty_server.Options{}
	if err := gotty_utils.ApplyDefaultValues(appOptions); err != nil {
		return errors.Wrap(err, "Gotty Configuration Failed: appOptions")
	}
	appOptions.Port = strconv.Itoa(freePort)
	appOptions.EnableBasicAuth = false
	appOptions.EnableTLSClientAuth = false
	appOptions.PermitWrite = true
	appOptions.WSOrigin = ".*?"
	appOptions.EnableReconnect = true
	appOptions.Term = "hterm"
	appOptions.Preferences = &gotty_server.HtermPrefernces{}
	if err := gotty_utils.ApplyDefaultValues(appOptions.Preferences); err != nil {
		return errors.Wrap(err, "Gotty Configuration Failed: appOptions.Preferences")
	}
	appOptions.Preferences.CtrlVPaste = true
	appOptions.Preferences.CursorColor = "rgba(255, 255, 255, 0.5)"

	backendOptions := &gotty_localcommand.Options{}
	if err := gotty_utils.ApplyDefaultValues(backendOptions); err != nil {
		return errors.Wrap(err, "Gotty Configuration Failed: backendOptions")
	}

	cmd := "bash"
	cmdArgs := []string{}
	factory, err := gotty_localcommand.NewFactory(cmd, cmdArgs, backendOptions)
	if err != nil {
		return errors.Wrap(err, "Gotty Configuration Failed: factory")
	}
	hostname, _ := os.Hostname()
	appOptions.TitleVariables = map[string]interface{}{
		"command":  cmd,
		"argv":     cmdArgs,
		"hostname": hostname,
	}
	srv, err := gotty_server.New(factory, appOptions)
	if err != nil {
		return errors.Wrap(err, "Gotty Server Init Failed")
	}
	go srv.Run(t.Ctx)

	var wtInfo models.GottyWebTerminalInfo
	wtInfo.GottyWebTerminalPort = freePort
	if err = task.SendObject(utils.StructToBytes(wtInfo), stream, 10*time.Second); err != nil {
		return err
	}

	return nil
}
