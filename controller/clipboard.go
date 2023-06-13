package controller

import (
	"context"
	"encoding/json"
	"fde_ctrl/response"
	"fde_ctrl/websocket"

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

func (impl ClipboardImpl) InitAndWatch() {
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
					websocket.Hub.Broadcast(websocket.WsResponse{Type: "clipboard", Data: message})
				}
			case data := <-imageCh:
				{
					message := clipReponse{
						Data:   data,
						Format: string(imageFormat),
					}
					info, _ := json.Marshal(message)
					websocket.Hub.Broadcast(websocket.WsResponse{Type: "clipboard", Data: info})
				}
			}
		}
	}()
}
