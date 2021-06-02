# joebot
Golang Command &amp; Control Server For Managing And Remote Accessing Machines Via Web Interface

## :raising_hand: Motivations

The big motivation is as follows.

- :tired_face: Hard to access into CI builders for troubleshooting issues which are not reproducible locally
- :tired_face: Tedious to forward particular port on CI builders for debugging
- :pray: Provide an easy way for accessing builder machines

## :cd: Installation

Directly download the binary from https://github.com/harmonicinc-com/joebot/releases

## :star: Features

- Web Terminal
- Web File Browser
- Web VNC (Default to port 5901)
- Dynamic Port Tunnelling

## Usage (Control Server)
```
$ joebot server --port=<Server_Port> --web-portal-port=<Server_Web_Portal_Port>
```

## Usage (Client)
```
$ joebot client --port=<Server_Port> --tag=customized-client-id <Server_IP>
```

### Web Interface
![Screenshot](https://raw.githubusercontent.com/harmonicinc-com/joebot/master/screenshot.PNG)

### Terminal Via Web Browser
![Screenshot](https://raw.githubusercontent.com/harmonicinc-com/joebot/master/screenshot-terminal.PNG)

### VNC Via Web Browser
![Screenshot](https://raw.githubusercontent.com/harmonicinc-com/joebot/master/screenshot-vnc.PNG)

### File Manager Via Web Browser
![Screenshot](https://raw.githubusercontent.com/harmonicinc-com/joebot/master/screenshot-filebrowser.gif)