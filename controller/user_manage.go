package controller

import (
	"fde_ctrl/logger"
	"os/exec"

	"github.com/gin-gonic/gin"
)

// a http request will call the user manager when openfde start
type UserManager struct {
	AppName string
}

func (impl UserManager) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.POST("/user_manager/unlock", impl.unlockHandler)
}

func (impl UserManager) unlockHandler(c *gin.Context) {
	userEventNotifier.Notify()
	return
}

func (impl *UserManager) Init(app string) {
	impl.AppName = app
	userEventNotifier.Register(impl.notify)
	return
}

type EventHandler func()

type EventNotifier struct {
	handlers []EventHandler
}

var userEventNotifier = EventNotifier{}

func (en *EventNotifier) Register(handler EventHandler) {
	en.handlers = append(en.handlers, handler)
}

func (en *EventNotifier) Notify() {
	for _, handler := range en.handlers {
		handler()
	}
}

var notifier = EventNotifier{}

func (impl *UserManager) notify() {
	if impl.AppName == "" {
		return
	}
	cmd := exec.Command("fde_launch", impl.AppName)
	err := cmd.Run()
	if err != nil {
		logger.Error("launch_app", impl.AppName, err)
	}
	impl.AppName = ""
}
