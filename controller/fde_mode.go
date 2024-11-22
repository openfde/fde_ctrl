package controller

import (
	"fde_ctrl/response"
	"fde_ctrl/windows_manager"

	"github.com/gin-gonic/gin"
)

type FDEModeCtrl struct {
}

func (impl FDEModeCtrl) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.GET("/fde_mode", impl.GetHandler)
}

type FDEModeRespons struct {
	FDEMode string
}

func (impl FDEModeCtrl) GetHandler(c *gin.Context) {
	response.Response(c, FDEModeRespons{FDEMode: string(windows_manager.GetFDEMode())})
}
