package client

import (
	"io/fs"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/asdine/storm"
	"github.com/harmonicinc-com/joebot/models"
	"github.com/spf13/afero"

	"github.com/filebrowser/filebrowser/v2/auth"
	"github.com/filebrowser/filebrowser/v2/diskcache"
	"github.com/filebrowser/filebrowser/v2/frontend"
	fbhttp "github.com/filebrowser/filebrowser/v2/http"
	"github.com/filebrowser/filebrowser/v2/img"
	"github.com/filebrowser/filebrowser/v2/rules"
	"github.com/filebrowser/filebrowser/v2/settings"
	"github.com/filebrowser/filebrowser/v2/storage/bolt"
	"github.com/filebrowser/filebrowser/v2/users"
	"github.com/harmonicinc-com/joebot/task"
	"github.com/harmonicinc-com/joebot/utils"
	"github.com/pkg/errors"
)

type FilebrowserTask struct {
	handleClient *Client
	*task.Task
}

func NewFilebrowserTask(client *Client) *FilebrowserTask {
	return &FilebrowserTask{
		client,
		task.NewTask(client.ctx, task.FilebrowserRequest, client.logger),
	}
}

func (t *FilebrowserTask) Handle(body []byte, stream net.Conn) error {
	if t.handleClient == nil {
		return errors.New("handleClient param not set")
	}

	freePort, err := t.handleClient.portsManager.ReservePort()
	if err != nil {
		return err
	}
	filebrowserServer, err := StartFilebrowserService(strconv.Itoa(freePort))
	if err != nil {
		return errors.Wrap(err, "Failed to create filebrowser server")
	}
	t.handleClient.filebrowserServer = filebrowserServer

	var fbInfo models.FilebrowserInfo
	fbInfo.FilebrowserPort = freePort
	fbInfo.DefaultDirectory = t.handleClient.FilebrowserDefaultDir
	if err = task.SendObject(utils.StructToBytes(fbInfo), stream, 10*time.Second); err != nil {
		return err
	}

	return nil
}

func StartFilebrowserService(port string) (*http.Server, error) {
	abs, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return nil, errors.New("Unable To Get Scope")
	}
	scope := strings.Split(abs, string(filepath.Separator))[0] + string(filepath.Separator)

	k, err := settings.GenerateKey()
	if err != nil {
		return nil, err
	}
	set := &settings.Settings{
		Key:           k,
		Signup:        false,
		CreateUserDir: false,
		Defaults: settings.UserDefaults{
			Scope:       scope,
			Locale:      "en",
			SingleClick: false,
			Perm: users.Permissions{
				Admin:    false,
				Execute:  true,
				Create:   true,
				Rename:   true,
				Modify:   true,
				Delete:   true,
				Share:    true,
				Download: true,
			},
		},
		AuthMethod: auth.MethodNoAuth,
		Branding:   settings.Branding{},
		Commands:   map[string][]string{},
		Shell:      []string{},
		Rules:      []rules.Rule{},
	}
	server := &settings.Server{
		Root:                  abs,
		BaseURL:               "",
		Socket:                "",
		Port:                  port,
		Log:                   "",
		TLSKey:                "",
		TLSCert:               "",
		Address:               "",
		EnableThumbnails:      false,
		ResizePreview:         false,
		EnableExec:            false,
		TypeDetectionByHeader: false,
	}

	imgSvc := img.New(1)

	var fileCache diskcache.Interface
	tmpDir, err := ioutil.TempDir("", "joebot")
	if err != nil {
		fileCache = diskcache.NewNoOp()
	} else {
		fileCache = diskcache.New(afero.NewOsFs(), tmpDir)
	}
	db, err := storm.Open(path.Join(tmpDir, "database"))
	if err != nil {
		return nil, errors.New("Failed To Create FileBrowser DB")
	}
	store, err := bolt.NewStorage(db)
	if err != nil {
		return nil, errors.New("Failed To Create FileBrowser DB Storage")
	}
	store.Settings.Save(set)
	store.Auth.Save(&auth.NoAuth{})
	store.Settings.SaveServer(server)

	user := &users.User{
		Username:     "admin",
		Password:     "admin",
		LockPassword: false,
	}
	set.Defaults.Apply(user)
	user.Perm.Admin = true
	store.Users.Save(user)

	assetsFs, err := fs.Sub(frontend.Assets(), "dist")
	if err != nil {
		return nil, errors.New("Failed To Create FileBrowser Assets FS")
	}

	handler, err := fbhttp.NewHandler(imgSvc, fileCache, store, server, assetsFs)
	if err != nil {
		return nil, errors.New("Failed To Create FileBrowser Handler")
	}

	srv := &http.Server{Addr: ":" + port, Handler: handler}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("Httpserver: ListenAndServe() error: %s", err)
		}
	}()
	return srv, nil
}
