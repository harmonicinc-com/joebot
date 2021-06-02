package utils

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func StructToBytes(e interface{}) []byte {
	var byteBuf bytes.Buffer
	err := gob.NewEncoder(&byteBuf).Encode(e)
	if err != nil {
		panic(err)
	}

	return byteBuf.Bytes()
}

func BytesToStruct(body []byte, structPtr interface{}) error {
	var byteBuf bytes.Buffer
	byteBuf.Write(body)

	return gob.NewDecoder(&byteBuf).Decode(structPtr)
}

// GetFreePort asks the kernel for a free open port that is ready to use.
//lbound and ubound specifies the target port range (inclusively)
//If lbound and ubound are both 0, then random ports would be used
func GetFreePort(lbound int, ubound int) (int, error) {
	targetPort := rand.Intn(ubound-lbound+1) + lbound

	addr, err := net.ResolveTCPAddr("tcp", "localhost:"+strconv.Itoa(targetPort))
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func IsPortOccupied(port int) bool {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:"+strconv.Itoa(port))
	if err != nil {
		return true
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return true
	}
	defer l.Close()
	return false
}

func IsFileExist(filePath string) bool {
	if _, err := os.Stat(filePath); err == nil {
		return true
	}
	return false
}

func WriteToFile(payload []byte, filePath string) error {
	return ioutil.WriteFile("file.txt", payload, 777)
}

func GetCurrentExecutablePath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}
	return dir
}

func Random(min, max int) int {
	rand.Seed(time.Now().UTC().UnixNano())
	return rand.Intn(max-min) + min
}
