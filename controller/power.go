package controller

import (
	"fde_ctrl/process_chan"
	"fde_ctrl/response"
	"os/exec"
	"os"
	"fde_ctrl/logger"
	"time"

	"github.com/gin-gonic/gin"
)

type PowerManager struct {
}

func (impl PowerManager) Setup(r *gin.RouterGroup) {
	v1 := r.Group("/v1")
	v1.POST("/power/off", impl.poweroffHandler)
	v1.POST("/power/logout", impl.logoutHandler)
	v1.POST("/power/restart", impl.restartHandler)
	v1.POST("/power/lock", impl.lockHandler)
	v1.POST("/power/sleep", impl.sleepHandler)
}
func (impl PowerManager)Init() {
	closeCh := make(chan struct{}, 2)
	openCh := make(chan struct{}, 2)
	_, err := os.Stat("/usr/bin/acpi_listen")
	if err != nil && os.IsNotExist(err) {
		logger.Warn("acpi_listen not found, lid events will not be handled")
		return 
	}
	go func() {
		cmd := exec.Command("acpi_listen")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return
		}
		if err := cmd.Start(); err != nil {
			return
		}
		defer cmd.Wait()

		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if err != nil {
				break
			}
			line := string(buf[:n])
			switch line {
			case "button/lid LID close\n":
				select {
				case closeCh <- struct{}{}:
				default:
				}
			case "button/lid LID open\n":
				select {
				case openCh <- struct{}{}:
				default:
				}
			}
		}
	}()

	go func() {
		for {
			<-openCh
			timer := time.NewTimer(4 * time.Second)
			select {
			case <-closeCh:
				timer.Stop()
				// Lid closed again, cancel sleep
			case <-timer.C:
				exec.Command("fde_fs", "-sleep").Run()
			}
		}
	}()
}
func (impl PowerManager) lockHandler(c *gin.Context) {
	exec.Command("dm-tool", "lock").Start()
	response.Response(c, nil)
}

func (impl PowerManager) logoutHandler(c *gin.Context) {
	process_chan.SendLogout()
	response.Response(c, nil)
}

func (impl PowerManager) poweroffHandler(c *gin.Context) {
	process_chan.SendPoweroff()
	response.Response(c, nil)
}

func (impl PowerManager) restartHandler(c *gin.Context) {
	process_chan.SendRestart()
	response.Response(c, nil)
}

func (impl PowerManager) sleepHandler(c *gin.Context) {
	cmd := exec.Command("fde_fs", "-sleep")
	go func() {
		time.Sleep(1 * time.Second)
		cmd.Run()
	}()
	response.Response(c, nil)
}
