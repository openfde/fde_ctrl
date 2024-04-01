package controller

import (
	"bufio"
	"fde_ctrl/logger"
	"fde_ctrl/response"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

type BrightNessManager struct {
	mode string
}

var __BUS string

func init() {
	cmd := exec.Command("fde_brightness", "-mode", "detect")
	output, err := cmd.StdoutPipe()
	if err != nil {
		os.Exit(1)
	}

	if err := cmd.Start(); err != nil {
		os.Exit(1)
	}

	scanner := bufio.NewScanner(output)

	for scanner.Scan() {
		__BUS = scanner.Text()
	}

	if err := scanner.Err(); err != nil {
		os.Exit(1)
	}

	if err := cmd.Wait(); err != nil {
		os.Exit(1)
	}
	return
}

func (impl BrightNessManager) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.GET("/brightness", impl.getHandler)
	v1.POST("/brightness", impl.setHandler)
}

type BrightnessResponse struct {
	Brightness    string
	MaxBrightness string
}

func (impl BrightNessManager) getHandler(c *gin.Context) {
	if len(__BUS) == 0 {
		response.Response(c, BrightnessResponse{Brightness: "80", MaxBrightness: "0"})
		return
	}
	cmd := exec.Command("fde_brightness", "-mode", "get", __BUS)
	output, err := cmd.StdoutPipe()
	if err != nil {
		logger.Error("stdoutpipe_brightness_process", nil, err)
		response.ResponseError(c, http.StatusInternalServerError, err)
		return
	}

	if err := cmd.Start(); err != nil {
		logger.Error("start_brightness_process", nil, err)
		response.ResponseError(c, http.StatusInternalServerError, err)
		return
	}

	scanner := bufio.NewScanner(output)
	var res BrightnessResponse
	for scanner.Scan() {
		line := scanner.Text()
		lines := strings.Fields(line)
		if len(lines) >= 2 {
			res.Brightness = lines[0]
			res.MaxBrightness = lines[1]
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("scanner_brightness_process", nil, err)
		response.ResponseError(c, http.StatusInternalServerError, err)
		return
	}

	if err := cmd.Wait(); err != nil {
		logger.Error("wait_brightness_process", nil, err)
		response.ResponseError(c, http.StatusInternalServerError, err)
		return
	}
	response.Response(c, res)
	return
}

type setBrightnessRequest struct {
	Brightness string
}

func (impl BrightNessManager) setHandler(c *gin.Context) {
	var request setBrightnessRequest
	err := c.ShouldBind(&request)
	if err != nil {
		response.ResponseParamterError(c, err)
		logger.Error("parse_brightness_process_set", nil, err)
		return
	}
	cmd := exec.Command("fde_brightness", "-mode", "set", "-bus", __BUS, "-brightness", request.Brightness)

	if err := cmd.Start(); err != nil {
		response.ResponseError(c, http.StatusInternalServerError, err)
		logger.Error("start_brightness_process_set", nil, err)

		return
	}

	if err := cmd.Wait(); err != nil {
		response.ResponseError(c, http.StatusInternalServerError, err)
		logger.Error("wait_brightness_process_set", nil, err)
		return
	}
	response.Response(c, nil)
	return
}
