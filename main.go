package main

import (
	"context"
	"fde_ctrl/conf"
	"fde_ctrl/controller"
	"fde_ctrl/controller/middleware"
	"fde_ctrl/fdedroid"
	"fde_ctrl/ibus"
	"fde_ctrl/logger"
	"fde_ctrl/process_chan"
	"fde_ctrl/websocket"
	"fde_ctrl/windows_manager"

	"os/exec"

	// "io/ioutil"

	// "errors"

	"github.com/gin-gonic/gin"
	"github.com/godbus/dbus/v5"
)

const socket = "./fde_ctrl.sock"

func setup(r *gin.Engine, configure conf.Configure) error {

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
	var pm controller.PowerManager
	var dm controller.DisplayManager
	group := r.Group("/api")
	err := apps.Scan()
	if err != nil {
		return err
	}
	var controllers []controller.Controller
	clipboard.InitAndWatch(configure)
	dm.SetMirror()

	controllers = append(controllers, clipboard, pm, &apps, vnc, dm)
	for _, value := range controllers {
		value.Setup(group)
	}

	return nil
}

func main() {
	configure, err := conf.Read()
	if err != nil {
		logger.Error("read_conf", nil, err)
		return
	}
	mainCtx, mainCtxCancelFunc := context.WithCancel(context.Background())
	err = ibus.Start(mainCtx)
	if err != nil {
		logger.Error("start_ibus", nil, err)
		return
	}
	var cmds []*exec.Cmd
	//step 1 start kwin
	var cmdWinMan *exec.Cmd
	cmdWinMan, err = windows_manager.Start(mainCtx, configure.WindowsManager, mainCtxCancelFunc)
	if err != nil {
		logger.Error("start_windows_manager", configure.WindowsManager.Name, err)
		return
	}
	logger.Info("start_windows_manager", configure.WindowsManager)
	if cmdWinMan != nil {
		cmds = append(cmds, cmdWinMan)
	}
	var droid fdedroid.Fdedroid
	if configure.WindowsManager.IsWayland() {
		droid = new(fdedroid.Waydroid)
	} else {
		droid = new(fdedroid.Anbox)
	}
	cmdSession, err := droid.Start(mainCtx, mainCtxCancelFunc, configure)
	if err != nil {
		logger.Error("fdedroid_start", configure.WindowsManager.IsWayland(), err)
		killSonProcess(cmds)
		return
	}
	cmds = append(cmds, cmdSession)

	go websocket.SetupWebSocket()
	//scan app from linux
	engine := gin.New()
	engine.Use(middleware.LogHandler(), gin.Recovery())
	engine.Use(middleware.ErrHandler())
	if err := setup(engine, configure); err != nil {
		logger.Error("setup", nil, err)
		return
	}
	// 启动HTTP服务器
	go engine.Run(":18080")

	// conn, err := dbus.ConnectSessionBus()
	// if err != nil {
	// 	mainCancel()
	// 	fmt.Fprintln(os.Stderr, "Failed to connect to session bus:", err)
	// 	os.Exit(1)
	// }
	// defer conn.Close()
	// err = initDdusForSignal(conn)
	// if err != nil {
	// 	fmt.Fprintln(os.Stderr, "Failed to connect to session bus:", err)
	// 	return
	// }

	// signal := make(chan *dbus.Signal, 10)
	// conn.Signal(signal)
	// defer conn.RemoveSignal(signal)
	if mainCtx.Err() == nil {
		select {
		case <-mainCtx.Done():
			{
				logger.Info("context_done", "exit due to unexpected canceled context")
				killSonProcess(cmds)
				fdedroid.StopAndroidContainer(context.Background(), fdedroid.FDEContainerName)
				return
			}
		// case <-signal:
		// 	{
		// 		killSonProcess(cmds)
		// 		fmt.Println("exit due to some one send logout signal")
		// 		return
		// 	}
		case action := <-process_chan.ProcessChan:
			{
				killSonProcess(cmds)
				if configure.WindowsManager.IsWayland() {
					fdedroid.StopWaydroidContainer(context.Background())
				} else {
					fdedroid.StopAndroidContainer(context.Background(), fdedroid.FDEContainerName)
				}
				switch action {
				case process_chan.Restart:
					{
						logger.Info("restart", "exit due to some one send restart signal")
						cmd := exec.Command("restart")
						err = cmd.Run()
						if err != nil {
							logger.Error("restart_failed", nil, err)
						}
						return
					}
				case process_chan.Logout:
					{
						// logout
						logger.Info("logout", "exit due to some one send logout signal")
						return
					}
				case process_chan.Poweroff:
					{
						// poweroff
						logger.Info("power_off", "exit due to some one send poweroff signal")
						cmd := exec.Command("shutdown", "-h", "now")
						err = cmd.Run()
						if err != nil {
							logger.Error("shutdown_failed", nil, err)
						}
						return
					}
				}
			}
		}
	} else {
		logger.Error("main_ctx_error", nil, mainCtx.Err())
	}
}

func killSonProcess(cmds []*exec.Cmd) {
	for index, _ := range cmds {
		logger.Info("kill_son_process", index)
		cmds[index].Process.Kill()
		// cmds[index].Process.Wait()
	}
}

func initDdusForSignal(conn *dbus.Conn) error {
	return conn.AddMatchSignal(
		dbus.WithMatchObjectPath("/org/remoteAndroid/Dbus"),
		dbus.WithMatchInterface("org.remoteAndroid.Dbus"),
		dbus.WithMatchMember("Logout"))

}
