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

func (d *DbusNotify) Init() error {
	if d.isServerRunning() {
		return nil
	}
	os.Remove(unixSocketPath)
	var err error
	d.conn, err = dbus.ConnectSessionBus()
	if err != nil {
		logger.Error("connect_session_bus", nil, err)
		return err
	}
	err = d.startServer()
	if err != nil {
		logger.Error("create_local_unix_server", nil, err)
		return err
	}
	d.dbusChanl = make(chan string, 20)
	d.iface = "com.openfde.NotifyIf"
	d.path = dbus.ObjectPath("/com/openfde/Notify")
	d.signalName = "appStateNotify"
	d.startDbusMessageWorker()
	return nil
}

const dbusLocalServerCheck = "check"

func (d *DbusNotify) isServerRunning() bool {
	_, err := os.Stat(unixSocketPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	c, err := net.DialTimeout("unix", unixSocketPath, 500*time.Millisecond)
	defer c.Close()
	if err == nil {
		_, err = c.Write([]byte(dbusLocalServerCheck))
		if err != nil {
			return false
		}
		return true
	}
	if d.conn == nil {
		err := errors.New("DBus connection is not initialized")
		logger.Error("dbus_connection_not_initialized", nil, err)
		return false
	}
	return false
}

func (d *DbusNotify) SendDbusMessage(msg string) error {
	// 判断服务是否启动
	if d.isServerRunning() {
		c, err := net.DialTimeout("unix", unixSocketPath, 500*time.Millisecond)
		defer c.Close()
		if err == nil {
			c.Write([]byte(msg))
		}
		return nil
	}
	return errors.New("Server is not running")
}

func (d *DbusNotify) startDbusMessageWorker() {
	for {
		select {
		case msg := <-d.dbusChanl:
			sig := &dbus.Signal{
				Sender: ":1.0",
				Path:   d.path,
				Name:   d.iface + "." + d.signalName,
				Body:   []interface{}{msg},
			}
			d.conn.Emit(sig.Path, sig.Name, sig.Body...)
		default:
			return
		}
	}
}

func (d *DbusNotify) startServer() error {
	var err error
	d.localSock, err = net.Listen("unix", unixSocketPath)
	if err != nil {
		return err
	}
	go func() {
		for {
			conn, err := d.localSock.Accept()
			if err != nil {
				break
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				n, _ := c.Read(buf)
				cmd := string(buf[:n])
				if cmd == dbusLocalServerCheck {

				} else {
					d.dbusChanl <- cmd
				}
				logger.Info("send dbus message", string(buf[:n]))
			}(conn)
		}
	}()
	defer func() {
		d.localSock.Close()
		os.Remove(unixSocketPath)
	}()
	return nil
}
