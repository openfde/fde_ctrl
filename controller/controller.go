package controller

import (
	"fde_ctrl/conf"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, app string, configure conf.Configure) {

	var vnc VncAppImpl
	vnc.Conf = configure
	var apps Apps
	var pm PowerManager
	var xserver XserverAppImpl
	var fdeModeCtrl FDEModeCtrl
	xserver.Conf = configure
	var brightness BrightNessManager
	fsfusing := FsFuseManager{}
	fsfusing.Init()
	group := r.Group("/api")
	apps.Scan(configure)
	var userManager *UserManager
	userManager.Init(app)
	var controllers []Controller
	controllers = append(controllers, pm, &apps, vnc, xserver, brightness, fsfusing, fdeModeCtrl, userManager)
	for _, value := range controllers {
		value.Setup(group)
	}

	return
}

type Controller interface {
	Setup(r *gin.RouterGroup)
}
