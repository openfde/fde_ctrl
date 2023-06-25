package main

import (
	"context"
	"fde_ctrl/controller"
	"fde_ctrl/logger"
	"fde_ctrl/middleware"
	"fde_ctrl/process_chan"
	"fde_ctrl/websocket"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-ini/ini"
	"github.com/godbus/dbus/v5"
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
	var pm controller.PowerManager
	group := r.Group("/api")
	err := apps.Scan()
	if err != nil {
		return err
	}
	clipboard.InitAndWatch()
	pm.Setup(group)
	clipboard.Setup(group)
	apps.Setup(group)
	vnc.Setup(group)

	return nil
}

const FDEDaemon = "fde_session"

func main() {
	cfg, err := ini.Load("/etc/fde.conf")
	if err != nil {
		logger.Error("load config", nil, err)
		return
	}

	// 获取配置文件中的值
	sectionAndroid := cfg.Section("Android")
	image := sectionAndroid.Key("Image").String()
	if len(image) == 0 {
		image = "fde:latest"
	}

	sectionHttp := cfg.Section("Http Server")
	hostIP := sectionHttp.Key("Host").String()
	if len(hostIP) == 0 {
		hostIP = "172.17.0.1"
	}

	mainCtx, _ := context.WithCancel(context.Background())

	//step 1 start kwin
	var cmdKwin *exec.Cmd
	_, exist := processExists("kwin")
	if !exist {
		//step 1 start kwin to enable windows manager
		cmdKwin = exec.CommandContext(mainCtx, "kwin")
		err = cmdKwin.Start()
		if err != nil {
			logger.Error("start_kwin", nil, err)
			return
		}
	}

	//step 2 stop kylin docker
	stopAndroidContainer(mainCtx, "kmre-1000-phytium")

	//step 3 start anbox hostside
	var cmdFdeDaemon *exec.Cmd
	_, exist = processExists(FDEDaemon)
	if !exist {
		//stop fdedroid
		err = stopAndroidContainer(mainCtx, FDEContainerName)
		if err != nil {
			logger.Error("start_fdedaemon_stop_fdedroid", nil, err)
			return
		}
		cmdFdeDaemon = exec.CommandContext(mainCtx, FDEDaemon, "session-manager", "--single-window", "--window-size=1920,1080",
			"-standalone", "--experimental")
		cmdFdeDaemon.Env = append(os.Environ())
		var stdout, stderr io.ReadCloser
		stdout, err = cmdFdeDaemon.StdoutPipe()
		if err != nil {
			logger.Error("stdout pipe for fde session", nil, err)
			return
		}
		stderr, err = cmdFdeDaemon.StderrPipe()
		if err != nil {
			logger.Error("stdout pipe for fde session", nil, err)
			return
		}

		err = cmdFdeDaemon.Start()
		if err != nil {
			logger.Error("start_fdedaemon", nil, err)
			return
		}
		output, err := ioutil.ReadAll(io.MultiReader(stdout, stderr))
		if err != nil {
			logger.Error("read start fde session failed", nil, err)
		}
		logger.Info("debug_fde_session", output)
	}

	//step 4  start fde android container
	err = startAndroidContainer(mainCtx, image, hostIP)
	if err != nil {
		logger.Error("start_android", nil, err)
		return
	}

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
				logger.Info("context_done", "exit due to canceled context")
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
				if action == process_chan.Logout {
					//logout
					logger.Info("logout", "exit due to some one send logout signal")
					return
				} else {
					//poweroff
					logger.Info("power_off", "exit due to some one send poweroff signal")
					stopAndroidContainer(mainCtx, FDEContainerName)
					var cmds []*exec.Cmd
					cmds = append(cmds, cmdFdeDaemon, cmdKwin)
					killSonProcess(cmds)
					//TODO call poweroff
					return
				}
			}
		}
	}
}

func killSonProcess(cmds []*exec.Cmd) {
	for index, _ := range cmds {
		cmds[index].Process.Kill()
		cmds[index].Process.Wait()
	}
}

func initDdusForSignal(conn *dbus.Conn) error {
	return conn.AddMatchSignal(
		dbus.WithMatchObjectPath("/org/remoteAndroid/Dbus"),
		dbus.WithMatchInterface("org.remoteAndroid.Dbus"),
		dbus.WithMatchMember("Logout"))

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
