package handler

import (
	"context"
	"encoding/binary"
	"net"
	"time"

	"github.com/harmonicinc-com/joebot/task"
	"github.com/hashicorp/yamux"
	errors "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type IncomingRequestHandler struct {
	logger *logrus.Logger
	ctx    context.Context

	session  *yamux.Session
	handlers map[task.TaskType](task.HandlerFunc)

	errorCallback func(error)
}

func NewIncomingRequestHandler(ctx context.Context, session *yamux.Session, logger *logrus.Logger) *IncomingRequestHandler {
	if logger == nil {
		logger = logrus.New()
	}

	handler := &IncomingRequestHandler{}
	handler.logger = logger
	handler.ctx = ctx
	handler.session = session
	handler.handlers = make(map[task.TaskType](task.HandlerFunc))

	return handler
}

func (handler *IncomingRequestHandler) OnError(callback func(error)) {
	handler.errorCallback = callback
}

func (handler *IncomingRequestHandler) handleError(err error) {
	if handler.errorCallback != nil {
		handler.errorCallback(err)
	}
}

func (handler *IncomingRequestHandler) RegisterTask(task task.Tasker) {
	handler.handlers[task.GetType()] = task.Handle
}

func (handler *IncomingRequestHandler) Start() {
	go func() {
		for {
			select {
			case <-handler.ctx.Done():
				handler.logger.Info("Stop Client Connection Handler")
				return
			default:
				stream, err := handler.session.Accept()
				if err != nil {
					err = errors.Wrap(err, "Requests Handler: Unable to create yamux stream")
					handler.handleError(err)
					continue
				}

				go func(stream net.Conn) {
					defer stream.Close()

					var reqType task.TaskType
					var reqBodyLen uint64 //number of bytes
					var reqBody []byte

					buf := make([]byte, 4)
					stream.SetReadDeadline(time.Now().Add(30 * time.Second))
					_, err = stream.Read(buf)
					if err != nil {
						err = errors.Wrap(err, "Unable to read request type from stream")
						handler.logger.Error(err)
						return
					}
					reqType = task.TaskType(binary.LittleEndian.Uint32(buf))

					buf = make([]byte, 8)
					stream.SetReadDeadline(time.Now().Add(30 * time.Second))
					_, err = stream.Read(buf)
					if err != nil {
						err = errors.Wrap(err, "Unable to read request body length from stream")
						handler.logger.Error(err)
						return
					}
					reqBodyLen = binary.LittleEndian.Uint64(buf)

					if reqBodyLen > 0 {
						reqBody = make([]byte, reqBodyLen)
						stream.SetReadDeadline(time.Time{})
						_, err = stream.Read(reqBody)
						if err != nil {
							err = errors.Wrap(err, "Unable to read request body from stream")
							handler.logger.Error(err)
							return
						}
					}

					if f, ok := handler.handlers[reqType]; ok {
						err := f(reqBody, stream)
						if err != nil {
							handler.logger.Error(errors.Wrap(err, "Request Handler Failed To Handle Request"))
						}
					} else {
						handler.logger.Error(errors.New("Request Handler Not Found"))
						return
					}
				}(stream)
			}
		}
	}()
}
