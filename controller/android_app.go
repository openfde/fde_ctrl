package controller

import (
	"fde_ctrl/response"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

type AndroidApp struct {
	Name        string
	PackageName string
}

type AndroidApps []AndroidApp

func (impl AndroidApp) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.GET("/android/apps", impl.AppsHandler)
}

func (impl AndroidApp) AppsHandler(c *gin.Context) {
	cmd := exec.Command("waydroid", "app", "list")
	output, err := cmd.Output()
	if err != nil {
		response.ResponseError(c, http.StatusInternalServerError, err)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var appsList AndroidApps
	var app AndroidApp
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name: ") {
			app.Name = strings.TrimPrefix(line, "Name: ")
		} else if strings.HasPrefix(line, "packageName: ") {
			app.PackageName = strings.TrimPrefix(line, "packageName: ")
			appsList = append(appsList, app)
			app = AndroidApp{}
		}
	}
	response.Response(c, appsList)
	return
}
