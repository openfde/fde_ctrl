package main

import (
	"fde_ctrl/middleware"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"

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
	var vnc VncAppImpl

	group := r.Group("/api")
	apps.Setup(group)
	vnc.Setup(group)

	// // 启动 HTTP 服务器
	// err = server.Serve(listener)
	// if err != nil {
	// 	log.Fatal("Error starting server: ", err)
	// }
}

func main() {
	// configPath := os.Getenv("FDE_CONFIG")
	// if len(configPath) == 0 {
	// 	configPath = "/etc/fde_config"
	// }
	// cfg, err := ini.Load(configPath)
	// if err != nil {
	// 	logger.Error(context.Background(),"load_config",nil,err)
	// 	return
	// }
	// cfg.

	// mainCtx, _ := context.WithCancel(context.Background())

	//step 1 start kwin to enable windows manager
	go setupWebSocket()
	//scan app from linux
	engine := gin.New()
	engine.Use(middleware.LogHandler(), gin.Recovery())
	engine.Use(middleware.ErrHandler())
	setup(engine)
	// 启动HTTP服务器
	engine.Run(":18080")

}

func processExists(name string) (pid int, exist bool) {
	cmd := exec.Command("pgrep", name)
	output, err := cmd.Output()
	if err != nil {
		return pid, false
	}

	pid, err = strconv.Atoi(string(output[:len(output)-1]))
	if err != nil {
		return pid, false
	}
	cmd.Wait()
	return pid, true
}
