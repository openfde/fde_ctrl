package main

import (
	"fde_ctrl/controller"
	"fde_ctrl/logger"
	"fde_ctrl/middleware"
	"fde_ctrl/websocket"
	"os/exec"
	"strconv"

	"github.com/gin-gonic/gin"
)

const socket = "./fde_ctrl.sock"

func setup(r *gin.Engine) error {

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

	var vnc controller.VncAppImpl
	var apps controller.Apps
	var clipboard controller.ClipboardImpl
	group := r.Group("/api")
	err := apps.Scan()
	if err != nil {
		return err
	}
	clipboard.InitAndWatch()

	clipboard.Setup(group)
	apps.Setup(group)
	vnc.Setup(group)

	// // 启动 HTTP 服务器
	// err = server.Serve(listener)
	// if err != nil {
	// 	log.Fatal("Error starting server: ", err)
	// }
	return nil
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
	go websocket.SetupWebSocket()
	//scan app from linux
	engine := gin.New()
	engine.Use(middleware.LogHandler(), gin.Recovery())
	engine.Use(middleware.ErrHandler())
	if err := setup(engine); err != nil {
		logger.Error("setup", nil, err)
		return
	}
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
