package controller

import (
	"fde_ctrl/logger"
	"fde_ctrl/response"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type AndroidApp struct {
	Name        string `json:"name"`
	PackageName string `json:"packageName"` // package name of the app, like com.android.app
	IconPath    string `json:"icon"`        // path to the icon file
	Path        string `json:"path"`        // how to launch the app fde_launch com.android.app
	Uninstll    string `json:"uninst"`      // how to uninstall the app fde_uninstall com.android.app
}

type AndroidAppsResponse struct {
	Apps []AndroidApp `json:"app info list"`
}

var fdeAppIconBaseDir = ".local/share/openfde/icons"

type AndroidApps []AndroidApp

func (impl AndroidApp) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.GET("/android/apps", impl.AppsHandler)
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

func (impl AndroidApp) AppsHandler(c *gin.Context) {
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
