package controller

import (
	"fde_ctrl/conf"
	"fde_ctrl/logger"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, app string, configure conf.Configure) {

	var vnc VncAppImpl
	vnc.Conf = configure
	var linuxApps LinuxApps
	var pm PowerManager
	var xserver XserverAppImpl
	var fdeModeCtrl FDEModeCtrl
	xserver.Conf = configure
	var brightness BrightNessManager
	var AndroidAppCtrl AndroidAppCtrl
	var appNotify AppNotify
	fsfusing := FsFuseManager{}
	AndroidAppCtrl.Init()
	fsfusing.Init()
	appNotify.Init()
	group := r.Group("/api")
	logger.Info("gy_linux_app_scan", "hello")
	linuxApps.Scan()
	userManager := UserManager{}
	userManager.Init(app)
	var controllers []Controller
	controllers = append(controllers, pm, linuxApps, vnc, xserver, brightness, fsfusing, fdeModeCtrl, userManager, &AndroidAppCtrl, appNotify)
	for _, value := range controllers {
		value.Setup(group)
	}

	return
}

type Controller interface {
	Setup(r *gin.RouterGroup)
}
