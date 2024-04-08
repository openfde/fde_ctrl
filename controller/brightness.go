package controller

import (
	"errors"
	"fde_ctrl/logger"
	"fde_ctrl/response"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

type BrightNessManager struct {
	mode string
}

const (
	BrightnessErrorBusInvalid = 2
)

var __BUS []string

func detect() {
	cmd := exec.Command("fde_brightness", "-mode", "detect")
	output, err := cmd.Output()
	if err != nil {
		logger.Error("brightness_detect", nil, err)
	}
	if strings.Compare("sys", string(output)) == 0 {
		__BUS = []string{"sys"}
	} else {
		__BUS = strings.Split(string(output), ",")
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
	if len(__BUS) == 0 {
		err := errors.New("i2c bus has not been detected")
		logger.Error("get_brightness_empty_bus", nil, err)
		go detect()
		response.ResponseError(c, http.StatusPreconditionFailed, err)
		return
	}
	cmd := exec.Command("fde_brightness", "-mode", "get", "-bus", __BUS[0])
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

func (impl BrightNessManager) set(brightness, bus string) error {
	cmd := exec.Command("fde_brightness", "-mode", "set", "-bus", bus, "-brightness", brightness)
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == BrightnessErrorBusInvalid {
				go detect()
			}
		}
		logger.Error("run_brightness_process_set", brightness, err)
		return err
	}
	return nil
}

func (impl BrightNessManager) setHandler(c *gin.Context) {
	var request setBrightnessRequest
	err := c.ShouldBindJSON(&request)
	if err != nil {
		response.ResponseParamterError(c, err)
		logger.Error("parse_brightness_process_set", nil, err)
		return
	}
	if len(request.Brightness) == 0 {
		err := errors.New("Birghtness invalid")
		response.ResponseParamterError(c, err)
		logger.Error("brightness_set_para_checking", nil, err)
		return
	}
	if len(__BUS) == 0 {
		err = errors.New("i2c bus has not been detected")
		logger.Error("set_brightness_empty_bus", nil, err)
		go detect()
		response.ResponseError(c, http.StatusPreconditionFailed, err)
		return
	}
	for _, bus := range __BUS {
		err = impl.set(request.Brightness, bus)
		if err != nil {
			response.ResponseError(c, http.StatusInternalServerError, err)
			return
		}
	}
	response.Response(c, nil)
	return
}
