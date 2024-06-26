package controller

import (
	"errors"
	"fde_ctrl/logger"
	"fde_ctrl/response"
	"net/http"
	"os/exec"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type BrightNessManager struct {
}

const (
	BrightnessErrorBusInvalid = 2
)

var __BUS []string

var lock sync.Mutex

func detect() {
	lock.Lock()
	defer lock.Unlock()
	cmd := exec.Command("fde_brightness", "-mode", "detect")
	output, err := cmd.Output()
	if err != nil {
		logger.Error("brightness_detect", nil, err)
	}
	if strings.Compare("sys", string(output)) == 0 {
		__BUS = []string{"sys"}
	} else {
		current := strings.ReplaceAll(string(output), "\n", "")
		__BUS = strings.Split(current, ",")
	}
	return
}

func (impl BrightNessManager) detectHandler(c *gin.Context) {
	detect()
}

func (impl BrightNessManager) Setup(r *gin.RouterGroup) {
	go detect()
	v1 := r.Group("/v1")
	v1.GET("/brightness", impl.getHandler)
	v1.POST("/brightness/detect", impl.detectHandler)
	v1.POST("/brightness", impl.setHandler)
}

type BrightnessResponse struct {
	Brightness    string
	MaxBrightness string
}

func (impl BrightNessManager) getHandler(c *gin.Context) {
	lock.Lock()
	if len(__BUS) == 0 {
		err := errors.New("i2c bus has not been detected")
		logger.Error("get_brightness_empty_bus", nil, err)
		go detect()
		response.ResponseError(c, http.StatusPreconditionFailed, err)
		lock.Unlock()
		return
	}
	bus := __BUS[0]
	lock.Unlock()
	cmd := exec.Command("fde_brightness", "-mode", "get", "-bus", bus)
	var res BrightnessResponse

	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == BrightnessErrorBusInvalid {
				go detect()
			}
		}
		logger.Error("get_brightness_exec", nil, err)
		response.ResponseError(c, http.StatusInternalServerError, err)
		return
	}
	lines := strings.Fields(string(output))
	if len(lines) >= 2 {
		res.Brightness = lines[0]
		res.MaxBrightness = lines[1]
	} else {
		err = errors.New("output is invalid")
		logger.Error("get_brightness", lines, err)
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
	err := c.ShouldBindJSON(&request)
	if err != nil {
		response.ResponseParamterError(c, err)
		logger.Error("parse_brightness_process_set", nil, err)
		return
	}
	logger.Info("set_brightness", request)
	if len(request.Brightness) == 0 {
		err := errors.New("Birghtness invalid")
		response.ResponseParamterError(c, err)
		logger.Error("brightness_set_para_checking", nil, err)
		return
	}
	lock.Lock()
	if len(__BUS) == 0 {
		err = errors.New("i2c bus has not been detected")
		logger.Error("set_brightness_empty_bus", nil, err)
		go detect()
		response.ResponseError(c, http.StatusPreconditionFailed, err)
		lock.Unlock()
		return
	}
	buses := make([]string, len(__BUS))
	copy(buses, __BUS)
	lock.Unlock()
	for _, bus := range buses {
		cmd := exec.Command("fde_brightness", "-mode", "set", "-bus", bus, "-brightness", request.Brightness)
		err := cmd.Run()
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				if exitError.ExitCode() == BrightnessErrorBusInvalid {
					go detect()
					response.ResponseError(c, http.StatusPreconditionFailed, err)
					return
				}
			}
			logger.Error("run_brightness_process_set", request.Brightness, err)
			response.ResponseError(c, http.StatusInternalServerError, err)
			return
		}
	}
	response.Response(c, nil)
	return
}
