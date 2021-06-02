package sshconnect

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"net"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func connect(user, password, host, key string, port int, cipherList []string, initSession bool) (*ssh.Client, *ssh.Session, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		client       *ssh.Client
		config       ssh.Config
		session      *ssh.Session
		err          error
	)
	// get auth method
	auth = make([]ssh.AuthMethod, 0)
	if key == "" {
		auth = append(auth, ssh.Password(password))
	} else {
		pemBytes, err := ioutil.ReadFile(key)
		if err != nil {
			return nil, nil, err
		}

		var signer ssh.Signer
		if password == "" {
			signer, err = ssh.ParsePrivateKey(pemBytes)
		} else {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(password))
		}
		if err != nil {
			return nil, nil, err
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}

	if len(cipherList) == 0 {
		config = ssh.Config{
			Ciphers: []string{"aes128-ctr", "aes192-ctr", "aes256-ctr", "aes128-gcm@openssh.com", "arcfour256", "arcfour128", "aes128-cbc", "3des-cbc", "aes192-cbc", "aes256-cbc"},
		}
	} else {
		config = ssh.Config{
			Ciphers: cipherList,
		}
	}

	clientConfig = &ssh.ClientConfig{
		User:    user,
		Auth:    auth,
		Timeout: 30 * time.Second,
		Config:  config,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	// connet to ssh
	addr = fmt.Sprintf("%s:%d", host, port)

	if client, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, nil, err
	}

	if initSession {
		// create session
		if session, err = client.NewSession(); err != nil {
			return nil, nil, err
		}

		modes := ssh.TerminalModes{
			ssh.ECHO:          0,     // disable echoing
			ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
			ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		}

		if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
			return nil, nil, err
		}

		return client, session, nil
	} else {
		return client, nil, nil
	}

}

func UploadMyself(dstFilePath, username, password, host, key string, cmdlist []string, port, timeout int, cipherList []string, linuxMode bool, ch chan SSHResult) {
	chSSH := make(chan SSHResult)
	go func(username, password, host, key string, cmdlist []string, port int, cipherList []string, ch chan SSHResult) {
		conn, _, err := connect(username, password, host, key, port, cipherList, false)
		var sshResult SSHResult
		sshResult.Host = host

		if err != nil {
			sshResult.Success = false
			sshResult.Result = fmt.Sprintf("<%s>", err.Error())
			ch <- sshResult
			return
		}

		// create new SFTP client
		client, err := sftp.NewClient(conn)
		if err != nil {
			sshResult.Success = false
			sshResult.Result = fmt.Sprintf("<%s>", err.Error())
			ch <- sshResult
			return
		}
		defer client.Close()

		// create destination file
		dstFile, err := client.Create(dstFilePath)
		if err != nil {
			sshResult.Success = false
			sshResult.Result = fmt.Sprintf("<%s>", err.Error())
			ch <- sshResult
			return
		}
		defer dstFile.Close()

		curFile, err := filepath.Abs(os.Args[0])
		if err != nil {
			sshResult.Success = false
			sshResult.Result = fmt.Sprintf("<%s>", err.Error())
			ch <- sshResult
			return
		}

		// create source file
		srcFile, err := os.Open(curFile)
		if err != nil {
			sshResult.Success = false
			sshResult.Result = fmt.Sprintf("<%s>", err.Error())
			ch <- sshResult
			return
		}

		// copy source file to destination file
		bytes, err := io.Copy(dstFile, srcFile)
		if err != nil {
			sshResult.Success = false
			sshResult.Result = fmt.Sprintf("<%s>", err.Error())
			ch <- sshResult
			return
		}

		sshResult.Success = true
		sshResult.Result = fmt.Sprintf("<%d bytes copied>", bytes)
		ch <- sshResult
		return
	}(username, password, host, key, cmdlist, port, cipherList, chSSH)
	var res SSHResult

	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		res.Host = host
		res.Success = false
		res.Result = ("SSH run timeout：" + strconv.Itoa(timeout) + " second.")
		ch <- res
	case res = <-chSSH:
		ch <- res
	}
	return
}

func Dossh(username, password, host, key string, cmdlist []string, port, timeout int, cipherList []string, linuxMode bool, ch chan SSHResult) {
	chSSH := make(chan SSHResult)
	if linuxMode {
		go dossh_run(username, password, host, key, cmdlist, port, cipherList, chSSH)
	} else {
		go dossh_session(username, password, host, key, cmdlist, port, cipherList, chSSH)
	}
	var res SSHResult

	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		res.Host = host
		res.Success = false
		res.Result = ("SSH run timeout：" + strconv.Itoa(timeout) + " second.")
		ch <- res
	case res = <-chSSH:
		ch <- res
	}
	return
}

func dossh_session(username, password, host, key string, cmdlist []string, port int, cipherList []string, ch chan SSHResult) {
	_, session, err := connect(username, password, host, key, port, cipherList, true)
	var sshResult SSHResult
	sshResult.Host = host

	if err != nil {
		sshResult.Success = false
		sshResult.Result = fmt.Sprintf("<%s>", err.Error())
		ch <- sshResult
		return
	}
	defer session.Close()

	cmdlist = append(cmdlist, "exit")

	stdinBuf, _ := session.StdinPipe()

	var outbt, errbt bytes.Buffer
	session.Stdout = &outbt

	session.Stderr = &errbt
	err = session.Shell()
	if err != nil {
		sshResult.Success = false
		sshResult.Result = fmt.Sprintf("<%s>", err.Error())
		ch <- sshResult
		return
	}
	for _, c := range cmdlist {
		c = c + "\n"
		stdinBuf.Write([]byte(c))
	}
	session.Wait()
	if errbt.String() != "" {
		sshResult.Success = false
		sshResult.Result = errbt.String()
		ch <- sshResult
	} else {
		sshResult.Success = true
		sshResult.Result = outbt.String()
		ch <- sshResult
	}

	return
}

func dossh_run(username, password, host, key string, cmdlist []string, port int, cipherList []string, ch chan SSHResult) {
	_, session, err := connect(username, password, host, key, port, cipherList, true)
	var sshResult SSHResult
	sshResult.Host = host

	if err != nil {
		sshResult.Success = false
		sshResult.Result = fmt.Sprintf("<%s>", err.Error())
		ch <- sshResult
		return
	}
	defer session.Close()

	cmdlist = append(cmdlist, "exit")
	newcmd := strings.Join(cmdlist, "&&")

	var outbt, errbt bytes.Buffer
	session.Stdout = &outbt

	session.Stderr = &errbt
	err = session.Run(newcmd)
	if err != nil {
		sshResult.Success = false
		sshResult.Result = fmt.Sprintf("<%s>", err.Error())
		ch <- sshResult
		return
	}

	if errbt.String() != "" {
		sshResult.Success = false
		sshResult.Result = errbt.String()
		ch <- sshResult
	} else {
		sshResult.Success = true
		sshResult.Result = outbt.String()
		ch <- sshResult
	}

	return
}
