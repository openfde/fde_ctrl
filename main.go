package main

import (
	"fde_ctrl/middleware"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

const socket = "./fde_ctrl.sock"

func setupWebSocket() {
	h := newHub()
	go h.run()
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				return
			}
		}()
		h.handleWebSocket(w, r)
	})
	http.HandleFunc("/broadcast", func(w http.ResponseWriter, r *http.Request) {
		h.broadcastHandle(w, r)
	})

	err := http.ListenAndServe(":18081", nil)
	if err != nil {
		fmt.Println("Failed to start server:", err)
	}
}

func setup(r *gin.Engine) {

	// 创建 Unix Socket
	// os.Remove(socket)
	// listener, err := net.Listen("unix", socket)
	// if err != nil {
	// 	log.Fatal("Error creating socket: ", err)
	// }
	// defer listener.Close()
	// // 创建 HTTP 服务器
	// server := &http.Server{}

	// http.HandleFunc("/ws", handleWebSocket)

	var apps Apps

	group := r.Group("/api")
	apps.Setup(group)

	// // 启动 HTTP 服务器
	// err = server.Serve(listener)
	// if err != nil {
	// 	log.Fatal("Error starting server: ", err)
	// }
}

func main() {

	go setupWebSocket()
	//scan app from linux
	engine := gin.New()
	engine.Use(middleware.LogHandler(), gin.Recovery())
	engine.Use(middleware.ErrHandler())
	setup(engine)
	// 启动HTTP服务器
	engine.Run(":18080")
}
