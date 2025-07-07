package controller

import (
	"encoding/json"
	"fde_ctrl/logger"
	"fde_ctrl/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/godbus/dbus/v5"
)

type AppNotify struct {
	conn       *dbus.Conn
	path       dbus.ObjectPath
	iface      string
	signalName string
	dbusChanl  chan string
}

func (impl AppNotify) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.POST("/app_notify", impl.NotifyHandler)
}

type AppNotifyRequest struct {
	PackageName string
	OpCode      string
	Version     string
}

func (d *AppNotify) Init() error {
	var err error
	d.conn, err = dbus.ConnectSessionBus()
	if err != nil {
		logger.Error("connect_session_bus", nil, err)
		return err
	}
	d.dbusChanl = make(chan string, 20)
	d.iface = "com.openfde.NotifyIf"
	d.path = dbus.ObjectPath("/com/openfde/Notify")
	d.signalName = "appStateNotify"
	d.startDbusMessageWorker()
	return nil
}

func (d *AppNotify) SendDbusMessage(req AppNotifyRequest) error {
	requests, err := json.Marshal(req)
	if err != nil {
		logger.Error("marshal_app_notify_request", nil, err)
		return err
	}
	d.dbusChanl <- string(requests)
	return nil
}

func (d *AppNotify) startDbusMessageWorker() {
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

func (impl AppNotify) NotifyHandler(c *gin.Context) {
	var request AppNotifyRequest
	err := c.ShouldBind(&request)
	if err != nil {
		response.ResponseParamterError(c, err)
		return
	}
	err = impl.SendDbusMessage(request)
	if err != nil {
		logger.Error("send_dbus_message", nil, err)
		response.ResponseError(c, http.StatusBadRequest, err)
		return
	}

	response.Response(c, nil)
}
