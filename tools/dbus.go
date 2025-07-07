package tools

import (
	"errors"
	"fde_ctrl/logger"
	"net"
	"os"
	"time"

	dbus "github.com/godbus/dbus/v5"
)

const unixSocketPath = "/tmp/fde_ctrl_dbus.sock"

type DbusNotifyInterface interface {
	Init() error
	isServerRunning() bool
	startServer() (net.Listener, error)
	SendDbusMessage(msg string) error
}

type DbusNotify struct {
	dbusChanl  chan (string)
	conn       *dbus.Conn
	localSock  net.Listener
	iface      string
	path       dbus.ObjectPath
	signalName string
}

import (
	"bytes"
	"net/http"
)

const url = "http://127.0.0.1:18080/api/v1/app_notify"

func SendDbusMessage( msg string) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(msg)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to send http message, status: " + resp.Status)
	}
	return nil
}
