package logo 

import (
	"image"
	_ "image/png" // Register PNG decoder
	"fde_ctrl/logger"
	"log"
	"os"
	"fmt"
	"os/exec"
	"strconv"
	"errors"

	"github.com/nfnt/resize"
	"golang.org/x/sys/unix"

	"github.com/rajveermalviya/go-wayland/wayland/client"
	xdg_shell "github.com/rajveermalviya/go-wayland/wayland/stable/xdg-shell"
    cursor "github.com/rajveermalviya/go-wayland/wayland/cursor"
	plasmashell "fde_ctrl/plasmashell" // adjust to your actual module path
)

// Global app state
type appState struct {
	appID         string
	title         string
	pImage        *image.RGBA
	width, height int32
	frame         *image.RGBA
	exit          bool

	display     *client.Display
	registry    *client.Registry
	shm         *client.Shm
	compositor  *client.Compositor
	xdgWmBase   *xdg_shell.WmBase
	seat        *client.Seat
	seatVersion uint32

	surface     *client.Surface
	xdgSurface  *xdg_shell.Surface
	xdgTopLevel *xdg_shell.Toplevel

	// Plasma extension support
	plasmaShell   *plasmashell.OrgKdePlasmaShell
	plasmaSurface *plasmashell.OrgKdePlasmaSurface

	keyboard *client.Keyboard
	pointer  *client.Pointer
	cursorTheme   *cursor.Theme
	cursor        *cursor.Cursor
	cursorSurface *client.Surface

	pointerEvent pointerEvent

	// 多输出支持
	outputsByID    map[uint32]*client.Output              // 对象ID -> Output代理
	outputModes    map[uint32]struct{ w, h uint16 }       // 输出ID -> 当前模式尺寸
	enteredOutputs map[uint32]*client.Output              // 当前surface所在的输出（按ID索引）
}

type pointerEvent struct {
	surfaceX, surfaceY float64
	button, state      uint32
	serial             uint32
}

func setDensityForWayland(density int){
	cmd := exec.Command("fde_fs", "-density", strconv.Itoa(density))
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("set density failed", map[string]interface{}{
			"density": density,
			"output":  string(output),
		}, err)
	} else {
	    logger.Warn("set density %d success\n", density)
	}
}

func ShowWayland() {
	// 检查当前环境是否为 Wayland
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	if sessionType != "wayland" {
		logger.Error("unsupported_session", nil, errors.New("XDG_SESSION_TYPE is not wayland"))
		return
	}
	// 检查是否存在 DISPLAY 变量
	display := os.Getenv("WAYLAND_DISPLAY")
	logger.Warn("display is %s \n", display)
	if display == "" {
		logger.Error("error display", nil, errors.New("display not set"))
		os.Setenv("WAYLAND_DISPLAY", "wayland-0")
		//display = os.Getenv("WAYLAND_DISPLAY")
		//return
	}
	fileName := "/usr/share/backgrounds/openfde.png"

	pImage, err := rgbaImageFromFile(fileName)
	if err != nil {
		log.Fatal(err)
	}

	frameRect := pImage.Bounds()

	app := &appState{
		title:  fileName + " - imageviewer",
		appID:  "imageviewer",
		pImage: pImage,
		width:  int32(frameRect.Dx()),
		height: int32(frameRect.Dy()),
		frame:  pImage,
		outputsByID:    make(map[uint32]*client.Output),
		outputModes:    make(map[uint32]struct{ w, h uint16 }),
		enteredOutputs: make(map[uint32]*client.Output),
	}

	if err := app.initWindow(); err != nil {
        logger.Error("initWindow failed", nil, err)
        return
    }
	
	go func() {
		logoShowedWayland = true
		defer close(doneWayland)
		select {
		case <-doneWayland:
			// Clean up Wayland resources
			log.Println("closing")
			app.cleanup()
		}
	}()

	for !app.exit {
		app.dispatch()
	}
}

func rgbaImageFromFile(fileName string) (*image.RGBA, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	rgba, ok := img.(*image.RGBA)
	if !ok {
		bounds := img.Bounds()
		rgba = image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rgba.Set(x, y, img.At(x, y))
			}
		}
	}
	return rgba, nil
}

func (app *appState) initWindow() error {
	display, err := client.Connect("")
	if err != nil {
		logger.Error("unable to connect to wayland server", nil, err)
		return err
	}
	app.display = display

	display.SetErrorHandler(app.HandleDisplayError)

	registry, err := app.display.GetRegistry()
	if err != nil {
	    logger.Error("unable to get global registry object", nil, err)
		return err
	}
	app.registry = registry

	registry.SetGlobalHandler(app.HandleRegistryGlobal)

	// Multiple roundtrips to ensure all globals (including plasma_shell) are discovered
	app.displayRoundTrip()
	app.displayRoundTrip()
	app.displayRoundTrip()

	log.Println("all interfaces registered")

	surface, err := app.compositor.CreateSurface()
	if err != nil {
		logger.Error("unable to create compositor surface", nil, err)
		return err
	}
	app.surface = surface
	log.Println("created new wl_surface")

	// 设置 surface 的 enter/leave 事件，用于跟踪所在的输出
	surface.SetEnterHandler(app.HandleSurfaceEnter)
	surface.SetLeaveHandler(app.HandleSurfaceLeave)

	// Try to hide from taskbar using KDE plasma extension
	if app.plasmaShell != nil {
		plasmaSurface, err := app.plasmaShell.GetSurface(app.surface)
		if err != nil {
			logger.Error("failed to get org_kde_plasma_surface", nil, err)
		} else {
			app.plasmaSurface = plasmaSurface
			log.Println("obtained org_kde_plasma_surface")

			// Manually send set_skip_taskbar (opcode 5, skip=1)
			const opcodeSetSkipTaskbar = 5
			const reqLen = 12
			var buf [reqLen]byte
			l := 0
			client.PutUint32(buf[l:l+4], plasmaSurface.ID())
			l += 4
			client.PutUint32(buf[l:l+4], uint32(reqLen<<16|opcodeSetSkipTaskbar))
			l += 4
			client.PutUint32(buf[l:l+4], 1) // 1 = skip taskbar
			l += 4

			err = plasmaSurface.Context().WriteMsg(buf[:], nil)
			if err != nil {
				logger.Error("failed to send set_skip_taskbar", nil, err)
			} else {
				logger.Info("sent set_skip_taskbar(1)", "should hide from taskbar")
			}
		}
	} else {
		logger.Info("org_kde_plasma_shell not available", "cannot hide from taskbar")
	}

	xdgSurface, err := app.xdgWmBase.GetXdgSurface(surface)
	if err != nil {
		logger.Error("unable to get xdg_surface", nil, err)
		return err
	}
	app.xdgSurface = xdgSurface
	log.Println("got xdg_surface")

	xdgSurface.SetConfigureHandler(app.HandleSurfaceConfigure)

	xdgTopLevel, err := xdgSurface.GetToplevel()
	if err != nil {
		logger.Error("unable to get xdg_toplevel", nil, err)
		return err
	}
	app.xdgTopLevel = xdgTopLevel
	log.Println("got xdg_toplevel")

	xdgTopLevel.SetConfigureHandler(app.HandleToplevelConfigure)
	xdgTopLevel.SetCloseHandler(app.HandleToplevelClose)

	xdgTopLevel.SetTitle(app.title)
	xdgTopLevel.SetAppId(app.appID)

	xdgTopLevel.SetFullscreen(nil)

	app.surface.Commit()
	return nil
}

func (app *appState) dispatch() {
	app.display.Context().Dispatch()
}

func (app *appState) context() *client.Context {
	return app.display.Context()
}

func (app *appState) HandleRegistryGlobal(e client.RegistryGlobalEvent) {
	logger.Info("discovered", fmt.Sprintf("%q version %d", e.Interface, e.Version))

	switch e.Interface {
	case "wl_compositor":
		compositor := client.NewCompositor(app.context())
		err := app.registry.Bind(e.Name, e.Interface, e.Version, compositor)
		if err != nil {
			logger.Error("bind wl_compositor failed", nil, err)
		}
		app.compositor = compositor

	case "wl_shm":
		shm := client.NewShm(app.context())
		err := app.registry.Bind(e.Name, e.Interface, e.Version, shm)
		if err != nil {
			logger.Error("bind wl_shm failed", nil, err)
		}
		app.shm = shm
		shm.SetFormatHandler(app.HandleShmFormat)

	case "xdg_wm_base":
		wm := xdg_shell.NewWmBase(app.context())
		err := app.registry.Bind(e.Name, e.Interface, e.Version, wm)
		if err != nil {
			logger.Error("bind xdg_wm_base failed", nil, err)
		}
		app.xdgWmBase = wm
		wm.SetPingHandler(app.HandleWmBasePing)

	case "wl_seat":
		seat := client.NewSeat(app.context())
		err := app.registry.Bind(e.Name, e.Interface, e.Version, seat)
		if err != nil {
			logger.Error("bind wl_seat failed", nil, err)
		}
		app.seat = seat
		app.seatVersion = e.Version
		seat.SetCapabilitiesHandler(app.HandleSeatCapabilities)
		seat.SetNameHandler(app.HandleSeatName)

	case "org_kde_plasma_shell":
		if app.plasmaShell != nil {
			return
		}
		plasma := plasmashell.NewOrgKdePlasmaShell(app.context())
		err := app.registry.Bind(e.Name, e.Interface, e.Version, plasma)
		if err != nil {
			logger.Error("bind org_kde_plasma_shell failed", nil, err)
			return
		}
		app.plasmaShell = plasma
		log.Println("bound org_kde_plasma_shell")

	case "wl_output":
		output := client.NewOutput(app.context())
		err := app.registry.Bind(e.Name, e.Interface, e.Version, output)
		if err != nil {
			logger.Error("bind wl_output failed", nil, err)
			return
		}
		// 使用副本避免闭包捕获问题
		out := output
		// 按对象ID保存输出代理
		app.outputsByID[out.ID()] = out

		// 使用闭包为每个输出单独设置事件处理器
		out.SetGeometryHandler(func(e client.OutputGeometryEvent) {
			// 可选：记录几何信息，本例中忽略
		})

		out.SetModeHandler(func(e client.OutputModeEvent) {
			if e.Flags&uint32(client.OutputModeCurrent) != 0 {
				outputID := out.ID()
				app.outputModes[outputID] = struct{ w, h uint16 }{uint16(e.Width), uint16(e.Height)}
				logger.Info("output", fmt.Sprintf("%d mode (current): %dx%d", outputID, e.Width, e.Height))

				// 如果该输出当前已被 surface 进入，则立即更新全局屏幕尺寸
				if _, entered := app.enteredOutputs[outputID]; entered {
					screenWidthWayland = uint16(e.Width)
					screenHeightWayland = uint16(e.Height)
					logger.Info("surface is on this output", fmt.Sprintf("update global size to %dx%d", e.Width, e.Height))
				}
			}
		})

		out.SetDoneHandler(func(e client.OutputDoneEvent) {
			log.Println("output done")
		})

		out.SetScaleHandler(func(e client.OutputScaleEvent) {
			// 可选：处理缩放因子
		})

		log.Println("bound wl_output")
	}
}

func (app *appState) HandleShmFormat(e client.ShmFormatEvent) {
	logger.Info("supported", fmt.Sprintf("format: %v", client.ShmFormat(e.Format)))
}

func (app *appState) HandleSurfaceConfigure(e xdg_shell.SurfaceConfigureEvent) {
	app.xdgSurface.AckConfigure(e.Serial)

	buffer := app.drawFrame()
	app.surface.Attach(buffer, 0, 0)
	app.surface.Commit()
}

func (app *appState) HandleToplevelConfigure(e xdg_shell.ToplevelConfigureEvent) {
	width := int32(e.Width)
	height := int32(e.Height)

	if width == 0 || height == 0 {
		return
	}

	// 如果尺寸没变，就不处理
	if width == app.width && height == app.height {
		return
	}

	logger.Info("resize to", fmt.Sprintf("%dx%d", width, height))

	app.width = width
	app.height = height

	app.frame = app.createLetterboxedFrame()
}

// createLetterboxedFrame 创建保持原始宽高比、居中显示的画布（带黑边）
func (app *appState) createLetterboxedFrame() *image.RGBA {
	// 原图宽高比
	srcW := float64(app.pImage.Bounds().Dx())
	srcH := float64(app.pImage.Bounds().Dy())
	srcAspect := srcW / srcH

	// 目标屏幕宽高比
	dstW := float64(app.width)
	dstH := float64(app.height)
	dstAspect := dstW / dstH

	var drawW, drawH uint
	var offsetX, offsetY int

	if srcAspect > dstAspect {
		// 原图更宽 → 高度撑满，左右黑边
		drawH = uint(dstH)
		drawW = uint(dstH * srcAspect)
		offsetX = int((dstW - float64(drawW)) / 2)
		offsetY = 0
	} else {
		// 原图更高 → 宽度撑满，上下黑边
		drawW = uint(dstW)
		drawH = uint(dstW / srcAspect)
		offsetX = 0
		offsetY = int((dstH - float64(drawH)) / 2)
	}

	// 先创建和屏幕一样大的黑色画布
	canvas := image.NewRGBA(image.Rect(0, 0, int(dstW), int(dstH)))

	// 把原图等比缩放到 drawW × drawH
	scaled := resize.Resize(drawW, drawH, app.pImage, resize.Bilinear)

	// 把缩放后的图片画到 canvas 的居中位置
	for y := 0; y < scaled.Bounds().Dy(); y++ {
		for x := 0; x < scaled.Bounds().Dx(); x++ {
			canvas.Set(offsetX+x, offsetY+y, scaled.At(x, y))
		}
	}

	return canvas
}

func (app *appState) HandleToplevelClose(_ xdg_shell.ToplevelCloseEvent) {
	app.exit = true
}

func (app *appState) drawFrame() *client.Buffer {
	stride := app.width * 4
	size := int64(stride * app.height)

	file, err := createTempFile(size)
	if err != nil {
		logger.Error("create temp file failed", nil, err)
	}
	defer file.Close()

	data, err := unix.Mmap(int(file.Fd()), 0, int(size), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		logger.Error("mmap failed", nil, err)
	}
	defer unix.Munmap(data)

	pool, err := app.shm.CreatePool(int(file.Fd()), int32(size))
	if err != nil {
		logger.Error("create pool failed", nil, err)
	}
	defer pool.Destroy()

	buf, err := pool.CreateBuffer(0, app.width, app.height, stride, uint32(client.ShmFormatArgb8888))
	if err != nil {
		logger.Error("create buffer failed", nil, err)
	}

	copy(data, app.frame.Pix)
	bgra(data)

	buf.SetReleaseHandler(func(_ client.BufferReleaseEvent) {
		buf.Destroy()
	})

	return buf
}

func createTempFile(size int64) (*os.File, error) {
	fd, err := unix.MemfdCreate("shm", 0)
	if err != nil {
		return nil, err
	}
	if err := unix.Ftruncate(fd, size); err != nil {
		unix.Close(fd)
		return nil, err
	}
	return os.NewFile(uintptr(fd), "shm"), nil
}

func bgra(data []byte) {
	for i := 0; i < len(data); i += 4 {
		data[i], data[i+2] = data[i+2], data[i]
	}
}

// ────────────────────────────────────────────────
// Missing handlers ─ restored
// ────────────────────────────────────────────────

func (app *appState) displayRoundTrip() {
	cb, err := app.display.Sync()
	if err != nil {
		logger.Error("sync failed", nil, err)
	}
	defer cb.Destroy()

	done := false
	cb.SetDoneHandler(func(_ client.CallbackDoneEvent) {
		done = true
	})

	for !done {
		app.dispatch()
	}
}

func (app *appState) HandleDisplayError(e client.DisplayErrorEvent) {
	log.Fatalf("display error: %v", e)
}

func (app *appState) HandleWmBasePing(e xdg_shell.WmBasePingEvent) {
	app.xdgWmBase.Pong(e.Serial)
}

func (app *appState) HandleSeatCapabilities(e client.SeatCapabilitiesEvent) {
	if (e.Capabilities & uint32(client.SeatCapabilityPointer)) != 0 {
		if app.pointer == nil {
			app.attachPointer()
		}
	} else if app.pointer != nil {
		app.releasePointer()
	}

	if (e.Capabilities & uint32(client.SeatCapabilityKeyboard)) != 0 {
		if app.keyboard == nil {
			app.attachKeyboard()
		}
	} else if app.keyboard != nil {
		app.releaseKeyboard()
	}
}

func (app *appState) HandleSeatName(e client.SeatNameEvent) {
	logger.Info("seat", fmt.Sprintf("name: %s", e.Name))
}

func (app *appState) attachPointer() {
	ptr, err := app.seat.GetPointer()
	if err != nil {
		logger.Error("get pointer failed", nil, err)
	}
	app.pointer = ptr

	ptr.SetEnterHandler(app.HandlePointerEnter)
	ptr.SetLeaveHandler(app.HandlePointerLeave)
	ptr.SetMotionHandler(app.HandlePointerMotion)
	ptr.SetButtonHandler(app.HandlePointerButton)
	
	app.loadCursor()
}

func (app *appState) loadCursor() {
	// "" 表示使用系统默认主题，24 是光标大小（可改成 32）
	theme, err := cursor.LoadTheme("", 24, app.shm)
	if err != nil {
		logger.Error("failed to load cursor theme", nil, err)
		return
	}
	app.cursorTheme = theme

	// 获取标准箭头光标
	cur := theme.GetCursor("left_ptr")
	if cur == nil {
		cur = theme.GetCursor("default") // fallback
	}
	if cur == nil {
		logger.Error("no cursor found", nil, nil)
		return
	}
	app.cursor = cur

	// 创建用于显示光标的 surface
	surface, err := app.compositor.CreateSurface()
	if err != nil {
		logger.Error("failed to create cursor surface", nil, err)
		return
	}
	app.cursorSurface = surface

	log.Println("Custom cursor loaded successfully")
}

func (app *appState) releasePointer() {
	if app.pointer != nil {
		app.pointer.Release()
		app.pointer = nil
	}
}

func (app *appState) attachKeyboard() {
	kbd, err := app.seat.GetKeyboard()
	if err != nil {
		logger.Error("get keyboard failed", nil, err)
	}
	app.keyboard = kbd

	kbd.SetKeyHandler(func(e client.KeyboardKeyEvent) {
		if e.Key == 1 && e.State == uint32(client.KeyboardKeyStatePressed) { // ESC
			app.exit = true
		}
	})
}

func (app *appState) releaseKeyboard() {
	if app.keyboard != nil {
		app.keyboard.Release()
		app.keyboard = nil
	}
}

func (app *appState) HandlePointerEnter(e client.PointerEnterEvent) {
	app.pointerEvent.serial = e.Serial
	app.pointerEvent.surfaceX = float64(e.SurfaceX)
	app.pointerEvent.surfaceY = float64(e.SurfaceY)
	
	// 设置光标（关键部分）
	if app.pointer != nil && app.cursor != nil && app.cursorSurface != nil {
		images := app.cursor.Images
		if len(images) > 0 {
			img := images[0]

			// GetBuffer 返回 (*client.Buffer, error)
			buffer, err := img.GetBuffer()
			if err != nil || buffer == nil {
			    logger.Error("failed to get cursor buffer", nil, err)
				return
			}

			app.cursorSurface.Attach(buffer, 0, 0)
			app.cursorSurface.Commit()

			// HotspotX 和 HotspotY 是字段，不是方法
			app.pointer.SetCursor(e.Serial, app.cursorSurface,
				int32(img.HotspotX),
				int32(img.HotspotY))
		}
	}
}

func (app *appState) HandlePointerLeave(e client.PointerLeaveEvent) {
	// optional: clear position or serial if needed
}

func (app *appState) HandlePointerMotion(e client.PointerMotionEvent) {
	app.pointerEvent.surfaceX = float64(e.SurfaceX)
	app.pointerEvent.surfaceY = float64(e.SurfaceY)
}

func (app *appState) HandlePointerButton(e client.PointerButtonEvent) {
	app.pointerEvent.button = e.Button
	app.pointerEvent.state = e.State
	app.pointerEvent.serial = e.Serial
}

// ==================== wl_surface enter/leave 事件处理器 ====================

func (app *appState) HandleSurfaceEnter(e client.SurfaceEnterEvent) {
	outputID := e.Output.ID()
	app.enteredOutputs[outputID] = e.Output
	// 如果该输出的尺寸已知，则更新全局屏幕尺寸
	if mode, ok := app.outputModes[outputID]; ok {
		screenWidthWayland = mode.w
		screenHeightWayland = mode.h
		logger.Info("surface entered output", fmt.Sprintf("%d, size %dx%d", outputID, mode.w, mode.h))
	} else {
		logger.Info("surface entered output", fmt.Sprintf("%d but mode not yet known", outputID))
	}
}

func (app *appState) HandleSurfaceLeave(e client.SurfaceLeaveEvent) {
	outputID := e.Output.ID()
	delete(app.enteredOutputs, outputID)
	// 如果还有其它已进入的输出，切换到其中一个的尺寸
	if len(app.enteredOutputs) > 0 {
		for id := range app.enteredOutputs {
			if mode, ok := app.outputModes[id]; ok {
				screenWidthWayland = mode.w
				screenHeightWayland = mode.h
				logger.Info("surface left output", fmt.Sprintf("%d, now using output %d size %dx%d", outputID, id, mode.w, mode.h))
				break
			}
		}
	} else {
		// 没有进入任何输出，可以保留旧尺寸或置零，这里保持原样
		log.Println("surface left all outputs")
	}
}

// ==================== 清理 ====================

func (app *appState) cleanup() {
	if app.plasmaSurface != nil {
		app.plasmaSurface.Destroy()
	}
	if app.plasmaShell != nil {
		app.plasmaShell.Destroy()
	}

	if app.pointer != nil {
		app.releasePointer()
	}
	if app.keyboard != nil {
		app.releaseKeyboard()
	}

	if app.xdgTopLevel != nil {
		app.xdgTopLevel.Destroy()
	}
	if app.xdgSurface != nil {
		app.xdgSurface.Destroy()
	}
	if app.surface != nil {
		app.surface.Destroy()
	}
	if app.xdgWmBase != nil {
		app.xdgWmBase.Destroy()
	}
	if app.shm != nil {
		app.shm.Destroy()
	}
	if app.compositor != nil {
		app.compositor.Destroy()
	}
	
	for _, output := range app.outputsByID {
		output.Release()
	}
	if app.registry != nil {
		app.registry.Destroy()
	}
	
	if app.display != nil {
		app.display.Destroy()
		app.display = nil
	}
    if app.cursorSurface != nil {
		app.cursorSurface.Destroy()
		app.cursorSurface = nil
	}
	if app.cursorTheme != nil {
		app.cursorTheme = nil
	}
	if app.cursor != nil {
		app.cursor = nil
	}
	app.exit = true
}


var doneWayland = make(chan struct{})
var logoShowedWayland = false
var screenWidthWayland   uint16
var screenHeightWayland  uint16

func DisappearWayland() {
	if logoShowedWayland == false {
		return
	}
	logger.Warn(fmt.Sprintf("DisappearWayland screen size: %dx%d", screenWidthWayland, screenHeightWayland), nil)
	
	if screenWidthWayland <= 1920 {
		setDensityForWayland(160)
	} else {
		setDensityForWayland(256)
	}
	// 检查当前环境是否为 Wayland
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	if sessionType != "wayland" {
		return
	}

	// 检查是否存在 DISPLAY 变量
	display := os.Getenv("WAYLAND_DISPLAY")
	if display == "" {
		return
	}
	// 发送关闭信号
	doneWayland <- struct{}{}
}