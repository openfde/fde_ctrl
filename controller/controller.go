package controller

import (
	"fde_ctrl/conf"
	"fde_ctrl/windows_manager"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, app string, configure conf.Configure, mode windows_manager.FDEMode) {
	var linuxApps LinuxApps
	var pm PowerManager
	var xserver XserverAppImpl
	var fdeModeCtrl FDEModeCtrl
	xserver.Conf = configure
	var brightness BrightNessManager
	var AndroidAppCtrl AndroidAppCtrl
	var appNotify AppNotify
	var versionManager VersionController
	fsfusing := FsFuseManager{}
	AndroidAppCtrl.Init()
	fsfusing.Init()
	appNotify.Init()
	pm.Init(mode)
	group := r.Group("/api")
	linuxApps.Scan()
	userManager := UserManager{}
	userManager.Init(app)
	var controllers []Controller
	controllers = append(controllers, pm, linuxApps, xserver, brightness, fsfusing, fdeModeCtrl, userManager, &AndroidAppCtrl, appNotify, versionManager)
	for _, value := range controllers {
		value.Setup(group)
	}

	return
}

type Controller interface {
	Setup(r *gin.RouterGroup)
}
