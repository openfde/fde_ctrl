package controller

import (
	"errors"
	"fde_ctrl/conf"
	"fde_ctrl/logger"
	"fde_ctrl/response"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type AndroidAppCtrl struct {
	Started           bool
	PidSurfaceFlinger string
	Conf              conf.ModeConf
}

type AndroidApp struct {
	Name        string `json:"name"`
	PackageName string `json:"packageName"` // package name of the app, like com.android.app
	Version     string `json:"version"`     //version of the app like 1.0.0
	IconPath    string `json:"icon"`        // path to the icon file
	Path        string `json:"path"`        // how to launch the app fde_launch com.android.app
	Uninstll    string `json:"uninst"`      // how to uninstall the app fde_uninstall com.android.app
	Desktop     string `json:"desktop"`     // desktop file path
}

type AndroidAppsResponse struct {
	Apps []AndroidApp `json:"app info list"`
}

func (impl *AndroidAppCtrl) notify() {
	if !impl.Started {
		//set navigation_mode accdording to the mode of desktop or app_fusing
		if conf.NaviModeIsHidden(impl.Conf.NaviMode) {
			logger.Info("set_navi", "hidden_gesture_2") //gesture hidden 2
			cmd := exec.Command("fde_fs", "-navmode", "2", "-setnav")
			err := cmd.Run()
			if err != nil {
				logger.Error("set_navigation_mode", "2", err)
			}
		} else {
			logger.Info("set_navi", "normal_three_button_0") // three buttons 0
			cmd := exec.Command("fde_fs", "-navmode", "0", "-setnav")
			err := cmd.Run()
			if err != nil {
				logger.Error("set_navigation_mode", "0", err)
			}
		}
	}
	impl.PidSurfaceFlinger, _ = impl.getSurfaceFlingerPid()
	logger.Info("android_system_started", impl.PidSurfaceFlinger)
	impl.Started = true
}

func (impl *AndroidAppCtrl) Init() {
	impl.Conf, _ = conf.ReadModeConf()
	userEventNotifier.Register(impl.notify)
}

var fdeAppIconBaseDir = ".local/share/openfde/icons"

type AndroidApps []AndroidApp

func (impl *AndroidAppCtrl) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.GET("/android/apps", impl.ListAppsHandler)
	v1.GET("/android/status", impl.StatusHandler)
}

func scanAppInfo(lines []string, home string) AndroidApps {
	var appsList AndroidApps
	var app AndroidApp
	for index, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name: ") {
			app.Name = strings.TrimPrefix(line, "Name: ")
		} else if strings.HasPrefix(line, "packageName: ") {
			app.PackageName = strings.TrimPrefix(line, "packageName: ")
		} else if strings.HasPrefix(line, "version:") {
			app.Version = strings.TrimPrefix(line, "version: ")
			//check if the category is android.intent.category.LAUNCHER, drop the app if not
			if index+2 <= len(lines) {
				if strings.HasPrefix(lines[index+1], "categories:") {
					category := strings.TrimSpace(lines[index+2])
					if category != "android.intent.category.LAUNCHER" {
						continue
					}
				}
			}
			app.IconPath = filepath.Join(home, fdeAppIconBaseDir, app.PackageName+".png")
			app.Uninstll = "fde_utils remove " + app.PackageName
			app.Path = "fde_launch " + app.PackageName
			app.Desktop = filepath.Join(home, ".local/share/applications", app.PackageName+"_fde.desktop")
			logger.Info("scan_android", app.PackageName)
			_, err := os.Stat(app.IconPath) // check if the icon file exists
			if err != nil && os.IsNotExist(err) {
				logger.Error("stat_android_icon", app.PackageName, err)
				app = AndroidApp{}
				continue
			}
			appsList = append(appsList, app)
			app = AndroidApp{}
		}
	}
	return appsList
}

func (impl *AndroidAppCtrl) getSurfaceFlingerPid() (string, error) {
	cmd := exec.Command("ps", "-ef")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	var pid string
	for _, line := range lines {
		if strings.Contains(line, "/system/bin/surfaceflinger") && !strings.Contains(line, "grep") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				pid = fields[1]
				break
			}
		}
	}

	if pid == "" {
		return "", errors.New("surfaceflinger process not found")
	}

	return pid, nil
}

func (impl *AndroidAppCtrl) StatusHandler(c *gin.Context) {
	if !impl.Started {
		response.ResponseError(c, http.StatusServiceUnavailable, errors.New("android system not started completely"))
	} else {
		pid, err := impl.getSurfaceFlingerPid()
		if err != nil {
			logger.Error("get_surfaceflinger_pid", nil, err)
			response.ResponseError(c, http.StatusServiceUnavailable, errors.New("android system not started completely"))
			return
		}
		if pid != impl.PidSurfaceFlinger {
			//if the pid changed, it means the android system restarted
			response.ResponseError(c, http.StatusServiceUnavailable, errors.New("android system not started completely"))
			return
		}
		response.Response(c, nil)
	}
	return
}

func (impl *AndroidAppCtrl) ListAppsHandler(c *gin.Context) {
	if !impl.Started {
		c.JSON(http.StatusPreconditionRequired, nil)
		return
	}
	pid, err := impl.getSurfaceFlingerPid()
	if err != nil {
		logger.Error("get_surfaceflinger_pid", nil, err)
		c.JSON(http.StatusPreconditionRequired, nil)
		return
	}
	if pid != impl.PidSurfaceFlinger {
		//if the pid changed, it means the android system restarted
		logger.Info("surfaceflinger_pid_changed", nil)
		c.JSON(http.StatusPreconditionRequired, nil)
		return
	}
	cmd := exec.Command("waydroid", "app", "list")
	rawresponse := c.DefaultQuery("raw", "0")
	output, err := cmd.Output()
	if err != nil {
		response.ResponseError(c, http.StatusInternalServerError, err)
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		response.ResponseError(c, http.StatusInternalServerError, err)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	apps := scanAppInfo(lines, home)
	if rawresponse == "1" {
		c.JSON(http.StatusOK, AndroidAppsResponse{Apps: apps})
		return
	}
	response.Response(c, AndroidAppsResponse{Apps: apps})
	return
}
