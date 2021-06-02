package client

import (
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/harmonicinc-com/joebot/models"
	"github.com/harmonicinc-com/joebot/task"
	"github.com/harmonicinc-com/joebot/utils"
	"github.com/pkg/errors"
	"golang.org/x/net/websocket"
)

type NovncTask struct {
	handleClient *Client
	*task.Task
}

func NewNovncTask(client *Client) *NovncTask {
	return &NovncTask{
		client,
		task.NewTask(client.ctx, task.NovncRequest, client.logger),
	}
}

func (t *NovncTask) Handle(body []byte, stream net.Conn) error {
	if t.handleClient == nil {
		return errors.New("handleClient param not set")
	}

	var websocketInfo models.NovncWebsocketInfo
	err := utils.BytesToStruct(body, &websocketInfo)
	if err != nil {
		return errors.Wrap(err, "Unable to decode request body into NovncWebsocketInfo object")
	}
	if !utils.IsPortOccupied(websocketInfo.VncServerPort) {
		return errors.New("No VNC Server Listening On Port " + strconv.Itoa(websocketInfo.VncServerPort))
	}

	freePort, err := t.handleClient.portsManager.ReservePort()
	if err != nil {
		return err
	}
	websocketInfo.NovncWebsocketPort = freePort
	websocketServer, err := ServeVncViaWebsocket(":"+strconv.Itoa(websocketInfo.VncServerPort), ":"+strconv.Itoa(websocketInfo.NovncWebsocketPort))
	if err != nil {
		return errors.Wrap(err, "Failed to ServeVncViaWebsocket")
	}
	t.handleClient.novncWebsocketServer = websocketServer

	return task.SendObject(utils.StructToBytes(websocketInfo), stream, 10*time.Second)
}

func ServeVncViaWebsocket(vncServerAddr string, websocketAddr string) (*http.Server, error) {
	//https://github.com/novnc/novnc
	//https://github.com/novnc/websockify
	mux := websocket.Server{
		Handshake: func(config *websocket.Config, r *http.Request) error {
			// config.Protocol = []string{"binary"}

			r.Header.Set("Access-Control-Allow-Origin", "*")
			r.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
			return nil
		},
		Handler: func(wsconn *websocket.Conn) {
			conn, err := net.Dial("tcp", vncServerAddr)

			if err != nil {
				log.Println(err)
				wsconn.Close()

			} else {
				wsconn.PayloadType = websocket.BinaryFrame

				go io.Copy(conn, wsconn)
				go io.Copy(wsconn, conn)

				select {}
			}
		},
	}

	srv := &http.Server{Addr: websocketAddr, Handler: mux}
	// http.Handle("/websockify", mux)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("Httpserver: ListenAndServe() error: %s", err)
		}
	}()
	return srv, nil
}
