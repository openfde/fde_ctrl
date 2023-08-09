package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fde_ctrl/conf"
	"fde_ctrl/logger"
	"fde_ctrl/response"
	"fde_ctrl/websocket"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"golang.design/x/clipboard"
)

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
	var responseData clipReponse
	if formatClip == clipboard.FmtImage {
		responseData.Data = data
		responseData.Format = string(imageFormat)
	} else {
		responseData.Data = string(data)
		responseData.Format = string(txtFormat)
	}
	response.Response(c, responseData)
}

type clipReponse struct {
	Data   interface{}
	Format string
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

const clipboardWsType = "clipboard"

func (impl ClipboardImpl) InitAndWatch(configure conf.Configure) {
	if configure.WindowsManager.IsWayland() {
		path := "/tmp/.X11-unix/X1"
		waitCnt := 0
		for {
			if waitCnt > 60 {
				err := errors.New("tiemout for 60s")
				logger.Error("x11_display_timeout", nil, err)
				return
			}
			if _, err := os.Stat(path); os.IsNotExist(err) {
				time.Sleep(time.Second)
				waitCnt++
			} else {
				if isX11DisplayConnected(":1") {
					os.Setenv("DISPLAY", ":1")
					break
				} else {
					err := errors.New(":1 is not a running x11 server")
					logger.Error("x11_display_connected", nil, err)
					return
				}
			}
		}
	}
	err := clipboard.Init()
	if err != nil {
		panic(err)
	}
	txtCh := clipboard.Watch(context.TODO(), clipboard.FmtText)
	imageCh := clipboard.Watch(context.TODO(), clipboard.FmtImage)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				return
			}
		}()
		for {
			select {
			case data := <-txtCh:
				{
					message := clipReponse{
						Data:   string(data),
						Format: string(txtFormat),
					}
					websocket.Hub.Broadcast(websocket.WsResponse{Type: clipboardWsType, Data: message})
				}
			case data := <-imageCh:
				{
					message := clipReponse{
						Data:   data,
						Format: string(imageFormat),
					}
					info, _ := json.Marshal(message)
					websocket.Hub.Broadcast(websocket.WsResponse{Type: clipboardWsType, Data: info})
				}
			}
		}
	}()
}
