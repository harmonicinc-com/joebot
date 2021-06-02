package task

import (
	"context"
	"encoding/binary"
	"net"
	"strconv"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	TASK_COMPLETE uint16 = 888
	TASK_FAILED   uint16 = 444
)

type TaskType uint32

const (
	ClientInfoUpdateRequest TaskType = 1 + iota
	PortTunnelRequest
	SSHTunnelRequest
	NovncRequest
	GottyWebTerminalRequest
	FilebrowserRequest
)

type HandlerFunc func([]byte, net.Conn) error

type Tasker interface {
	GetType() TaskType
	SetRequestParam(param interface{}) Tasker
	SetHandleParam(param interface{}) Tasker
	Handle(body []byte, stream net.Conn) error
	Request(session *yamux.Session, payload []byte) (net.Conn, error)
}

type Task struct {
	Ctx    context.Context
	Type   TaskType
	Logger *logrus.Logger
}

func NewTask(ctx context.Context, taskType TaskType, logger *logrus.Logger) *Task {
	return &Task{
		Ctx:    ctx,
		Type:   taskType,
		Logger: logger,
	}
}

func (task *Task) GetType() TaskType {
	return task.Type
}

func (task *Task) SetHandleParam(param interface{}) Tasker {
	return task
}

func (task *Task) SetRequestParam(param interface{}) Tasker {
	return task
}

func (task *Task) Request(session *yamux.Session, payload []byte) (net.Conn, error) {
	stream, err := session.Open()
	if err != nil {
		return nil, errors.Wrap(err, "Session Open Failed...")
	}

	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, uint32(task.Type))
	_, err = stream.Write(bs)
	if err != nil {
		return nil, errors.Wrap(err, "Write task type failed")
	}

	bs = make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, uint64(len(payload)))
	_, err = stream.Write(bs)
	if err != nil {
		return nil, errors.Wrap(err, "Write task payload length failed")
	}

	if len(payload) > 0 {
		_, err = stream.Write(payload)
		if err != nil {
			return nil, errors.Wrap(err, "Write task payload failed")
		}
	}

	return stream, nil
}

func (task *Task) Handle(body []byte, stream net.Conn) error {
	return errors.New("The handle function is empty | Task Type: " + strconv.Itoa(int(task.Type)))
}

func ConfirmTaskComplete(stream net.Conn) error {
	resp := make([]byte, 2)
	binary.LittleEndian.PutUint16(resp, TASK_COMPLETE)
	_, err := stream.Write(resp)
	return err
}

func WaitTaskCompleteSignal(timeout time.Duration, stream net.Conn) error {
	buf := make([]byte, 2)
	stream.SetReadDeadline(time.Now().Add(timeout))
	_, err := stream.Read(buf)
	if err != nil {
		return errors.Wrap(err, "Failed To Receive Task Complete Confirmation From Client")
	}
	val := binary.LittleEndian.Uint16(buf)
	if val == TASK_COMPLETE {
		return nil
	}

	return errors.New("Recieved Task Failed Status From Client")
}

func ReceiveObject(stream net.Conn, timeout time.Duration) ([]byte, error) {
	buf := make([]byte, 8)
	stream.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err := stream.Read(buf)
	if err != nil {
		err = errors.Wrap(err, "ReceiveObject Unable to read request body length from stream")
		return nil, err
	}
	reqBodyLen := binary.LittleEndian.Uint64(buf)

	reqBody := make([]byte, reqBodyLen)
	stream.SetReadDeadline(time.Now().Add(timeout))
	_, err = stream.Read(reqBody)
	if err != nil {
		err = errors.Wrap(err, "ReceiveObject Unable to read request body from stream")
		return nil, err
	}

	return reqBody, nil
}

func SendObject(payload []byte, stream net.Conn, timeout time.Duration) error {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, uint64(len(payload)))
	stream.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err := stream.Write(bs)
	if err != nil {
		return errors.Wrap(err, "SendObject writes payload length failed")
	}

	stream.SetWriteDeadline(time.Now().Add(timeout))
	_, err = stream.Write(payload)
	if err != nil {
		return errors.Wrap(err, "SendObject writes payload failed")
	}

	return nil
}
