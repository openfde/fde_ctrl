package controller

import (
	"fde_ctrl/response"

	"github.com/gin-gonic/gin"
	"golang.design/x/clipboard"
)

type ClipboardInterface interface {
	Setup(r *gin.RouterGroup)
	Init()
}

type ClipboardImpl struct {
}

func (impl ClipboardImpl) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.POST("/clipboard", impl.WriteHandler)
	v1.GET("/clipboard", impl.ReadHandler)
}

type wirteClipBoardRequest struct {
	Format string
	Data   string
}

type clipboardFormat string

const (
	txtFormat   = clipboardFormat("text")
	imageFormat = clipboardFormat("image")
)

func (impl ClipboardImpl) ReadHandler(c *gin.Context) {
	formatQuery := c.Query("Format")
	var formatClip clipboard.Format
	switch {
	case formatQuery == string(imageFormat):
		{
			formatClip = clipboard.FmtImage
		}
	default:
		{
			formatClip = clipboard.FmtText
		}
	}
	data := clipboard.Read(formatClip)
	response.Response(c, data)
}

func (impl ClipboardImpl) WriteHandler(c *gin.Context) {
	var request wirteClipBoardRequest
	err := c.ShouldBind(&request)
	if err != nil {
		response.ResponseParamterError(c, err)
		return
	}
	var formatClip = clipboard.FmtText
	switch {
	case request.Format == string(imageFormat):
		{
			formatClip = clipboard.FmtImage
		}
	}
	clipboard.Write(formatClip, []byte(request.Data))
	response.Response(c, nil)
}

func (impl ClipboardImpl) Init() {
	err := clipboard.Init()
	if err != nil {
		panic(err)
	}
}
