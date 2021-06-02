//go:generate go get github.com/UnnoTed/fileb0x
//go:generate go run github.com/UnnoTed/fileb0x joebot-html.json

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/harmonicinc-com/joebot/client"
	"github.com/harmonicinc-com/joebot/joebot_html"
	"github.com/harmonicinc-com/joebot/models"
	"github.com/harmonicinc-com/joebot/server"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	app = kingpin.New("joebot", "Command & Control Server/Client For Managing Machines Via Web Interface")

	serverCommand = app.Command("server", "Server Mode")
	serverPort    = serverCommand.Flag("port", "Port For Listening Slave Machine, Default = 13579").Default("13579").Short('p').Int()
	webPortalPort = serverCommand.Flag("web-portal-port", "Port For The Web Portal, Default = 8080").Default("8080").Short('w').Int()
	username      = serverCommand.Flag("user", "Username for login the web portal").String()
	password      = serverCommand.Flag("pw", "Password for login the web portal").String()

	clientCommand                = app.Command("client", "Client Mode")
	cServerIP                    = clientCommand.Arg("ip", "Server IP").Required().String()
	cServerPort                  = clientCommand.Flag("port", "Server Port, Default=13579").Default("13579").Short('p').Int()
	cAllowedPortRangeLBound      = clientCommand.Flag("allowed-port-lower-bound", "Lower Bound Of Allowed Port Range").Default("0").Short('l').Int()
	cAllowedPortRangeUBound      = clientCommand.Flag("allowed-port-upper-bound", "Upper Bound Of Allowed Port Range").Default("65535").Short('u').Int()
	cTags                        = clientCommand.Flag("tag", "Tags").Strings()
	cFilebrowserDefaultDirectory = clientCommand.Flag("dir", "Filebrowser Default Directory, Default=/").Default("/").Short('f').String()
)

func main() {
	defer func() {
		fmt.Println("Ended")
	}()

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case serverCommand.FullCommand():
		s := server.NewServer(nil)
		s.Start(*serverPort)

		e := echo.New()
		v1 := e.Group("/api")

		if username != nil && password != nil && *username != "" && *password != "" {
			v1.Use(middleware.BasicAuth(func(user, pw string, c echo.Context) (bool, error) {
				if user == *username && pw == *password {
					return true, nil
				}
				return false, nil
			}))
		}

		v1.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{echo.GET, echo.HEAD, echo.PUT, echo.PATCH, echo.POST, echo.DELETE},
		}))
		e.GET("/", func(c echo.Context) error {
			b, err := joebot_html.ReadFile("index.html")
			if err != nil {
				log.Fatal(err)
			}
			return c.HTML(200, string(b))
		})
		e.GET("/*", echo.WrapHandler(joebot_html.Handler))
		v1.GET("/clients", func(c echo.Context) error {
			return c.JSON(http.StatusOK, s.GetClientsList())
		})
		v1.POST("/client/:id", func(c echo.Context) error {
			type msg struct {
				Message string `json:"message"`
			}

			client, err := s.GetClientById(c.Param("id"))
			if err != nil {
				return c.JSON(http.StatusNotFound, msg{err.Error()})
			}
			portStr := c.FormValue("target_client_port")
			port, err := strconv.Atoi(portStr)
			if err != nil || port <= 0 {
				return c.JSON(http.StatusBadRequest, msg{"Invalid target_client_port"})
			}

			portTunnelInfo, err := client.CreateTunnel(port)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, msg{err.Error()})
			}

			return c.JSON(http.StatusOK, portTunnelInfo)
		})
		v1.POST("/bulk-install", func(c echo.Context) error {
			json := models.BulkInstallInfo{}

			if err := c.Bind(&json); err != nil {
				return err
			}
			result, err := s.BulkInstallJoebot(json)
			if err != nil {
				return err
			}

			return c.String(http.StatusOK, result)
		})
		e.Start(":" + strconv.Itoa(*webPortalPort))
	case clientCommand.FullCommand():
		wg := &sync.WaitGroup{}
		wg.Add(1)
		c := client.NewClient(*cServerIP, *cServerPort, *cAllowedPortRangeLBound, *cAllowedPortRangeUBound, *cTags, nil)
		c.FilebrowserDefaultDir = *cFilebrowserDefaultDirectory
		c.Start()
		wg.Wait()
	}
}
