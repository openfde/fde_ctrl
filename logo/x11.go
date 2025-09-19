package logo

import (
	"encoding/binary"
	"errors"
	"fde_ctrl/logger"
	"os"
	"reflect"
	"time"
	"unsafe"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/render"
	"github.com/BurntSushi/xgb/shm"
	"github.com/BurntSushi/xgb/xproto"

	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// #include <sys/ipc.h>
// #include <sys/shm.h>
import "C"

func F64ToFixed(f float64) render.Fixed { return render.Fixed(f * 65536) }
func FixedToF64(f render.Fixed) float64 { return float64(f) / 65536 }

var formats = map[byte]struct {
	format    render.Directformat
	transform func(color.Color) uint32
}{
	32: {
		format: render.Directformat{
			RedShift:   16,
			RedMask:    0xff,
			GreenShift: 8,
			GreenMask:  0xff,
			BlueShift:  0,
			BlueMask:   0xff,
			AlphaShift: 24,
			AlphaMask:  0xff,
		},
		transform: func(color color.Color) uint32 {
			r, g, b, a := color.RGBA()
			return (a>>8)<<24 | (r>>8)<<16 | (g>>8)<<8 | (b >> 8)
		},
	},
	30: {
		/*
			// Alpha makes compositing unbearably slow.
			AlphaShift: 30,
			AlphaMask:  0x3,
		*/
		format: render.Directformat{
			RedShift:   20,
			RedMask:    0x3ff,
			GreenShift: 10,
			GreenMask:  0x3ff,
			BlueShift:  0,
			BlueMask:   0x3ff,
		},
		transform: func(color color.Color) uint32 {
			r, g, b, a := color.RGBA()
			return (a>>14)<<30 | (r>>6)<<20 | (g>>6)<<10 | (b >> 6)
		},
	},
}

func Show() {
	// 检查当前环境是否为 X11
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	if sessionType != "x11" {
		logger.Error("unsupported_session", nil, errors.New("XDG_SESSION_TYPE is not x11"))
		return
	}
	// 检查是否存在 DISPLAY 变量
	display := os.Getenv("DISPLAY")
	if display == "" {
		return
	}

	// Load a picture from the command line.
	f, err := os.Open("/usr/share/backgrounds/openfde.png")
	if err != nil {
		logger.Error("open_image", nil, err)
		return
	}
	defer f.Close()

	img, name, err := image.Decode(f)
	if err != nil {
		logger.Error("decode_image", nil, err)
		return
	}
	bounds1 := img.Bounds()

	// Miscellaneous X11 initialization.
	X, err := xgb.NewConn()
	if err != nil {
		logger.Error("new_x_connection", nil, err)
		return
	}

	if err := render.Init(X); err != nil {
		logger.Error("init_render", nil, err)
		return
	}

	setup := xproto.Setup(X)
	screen := setup.DefaultScreen(X)

	visual, depth := screen.RootVisual, screen.RootDepth

	// Only go for 10-bit when the picture can make use of that range.
	prefer30 := false
	switch img.(type) {
	case *image.Gray16, *image.RGBA64, *image.NRGBA64:
		prefer30 = true
	}

	// XXX: We don't /need/ alpha here, it's just a minor improvement--affects
	// the backpixel value. (And we reject it in 30-bit depth anyway.)
Depths:
	for _, i := range screen.AllowedDepths {
		for _, v := range i.Visuals {
			// TODO: Could/should check other parameters, e.g., the RGB masks.
			if v.Class != xproto.VisualClassTrueColor {
				continue
			}
			if i.Depth == 32 || i.Depth == 30 && prefer30 {
				visual, depth = v.VisualId, i.Depth
				if !prefer30 || i.Depth == 30 {
					break Depths
				}
			}
		}
	}

	format, ok := formats[depth]
	if !ok {
		logger.Error("unsupported_depth", nil, nil)
		return

	}

	mid, err := xproto.NewColormapId(X)
	if err != nil {
		logger.Error("new_colormap_id", nil, err)
		return
	}

	_ = xproto.CreateColormap(
		X, xproto.ColormapAllocNone, mid, screen.Root, visual)

	wid, err := xproto.NewWindowId(X)
	if err != nil {
		logger.Error("new_window_id", nil, err)
		return
	}

	screenWidth := screen.WidthInPixels
	screenHeight := screen.HeightInPixels

	// 计算居中位置
	windowWidth := uint16(bounds1.Dx())
	windowHeight := uint16(bounds1.Dy())
	centerX := int16((screenWidth - windowWidth) / 2)
	centerY := int16((screenHeight - windowHeight) / 2)

	// Border pixel and colormap are required when depth differs from parent.
	_ = xproto.CreateWindow(X, depth, wid, screen.Root,
		centerX, centerY, windowWidth, windowHeight, 0, xproto.WindowClassInputOutput,
		visual, xproto.CwBackPixel|xproto.CwBorderPixel|xproto.CwEventMask|
			xproto.CwColormap, []uint32{format.transform(color.Alpha{0x80}), 0,
			xproto.EventMaskStructureNotify | xproto.EventMaskExposure,
			uint32(mid)})

	_ = xproto.MapWindow(X, wid)

	// 设置窗口类型为桌面，去掉标题栏和任务栏
	wmWindowType := xproto.InternAtom(X, false, uint16(len("_NET_WM_WINDOW_TYPE")), "_NET_WM_WINDOW_TYPE")
	// wmWindowTypeDesktop := xproto.InternAtom(X, false, uint16(len("_NET_WM_WINDOW_TYPE_DESKTOP")), "_NET_WM_WINDOW_TYPE_DESKTOP")
	wmWindowTypeNormal := xproto.InternAtom(X, false, uint16(len("_NET_WM_WINDOW_TYPE_NORMAL")), "_NET_WM_WINDOW_TYPE_NORMAL")

	wmWindowTypeReply, _ := wmWindowType.Reply()
	wmWindowTypeNormalReply, _ := wmWindowTypeNormal.Reply()

	if wmWindowTypeReply != nil && wmWindowTypeNormalReply != nil {
		_ = xproto.ChangeProperty(X, xproto.PropModeReplace, wid, wmWindowTypeReply.Atom,
			xproto.AtomAtom, 32, 1, (*[4]byte)(unsafe.Pointer(&wmWindowTypeNormalReply.Atom))[:])
	}

	// 设置窗口显示在最上层
	wmState := xproto.InternAtom(X, false, uint16(len("_NET_WM_STATE")), "_NET_WM_STATE")
	wmStateAbove := xproto.InternAtom(X, false, uint16(len("_NET_WM_STATE_ABOVE")), "_NET_WM_STATE_ABOVE")

	wmStateReply, _ := wmState.Reply()
	wmStateAboveReply, _ := wmStateAbove.Reply()

	if wmStateReply != nil && wmStateAboveReply != nil {
		_ = xproto.ChangeProperty(X, xproto.PropModeReplace, wid, wmStateReply.Atom,
			xproto.AtomAtom, 32, 1, (*[4]byte)(unsafe.Pointer(&wmStateAboveReply.Atom))[:])
	}

	pformats, err := render.QueryPictFormats(X).Reply()
	if err != nil {
		logger.Error("query_pict_formats", nil, err)
		return
	}

	// Similar to XRenderFindVisualFormat.
	// The DefaultScreen is almost certain to be zero.
	var pformat render.Pictformat
	for _, pd := range pformats.Screens[X.DefaultScreen].Depths {
		// This check seems to be slightly extraneous.
		if pd.Depth != depth {
			continue
		}
		for _, pv := range pd.Visuals {
			if pv.Visual == visual {
				pformat = pv.Format
			}
		}
	}

	// Wrap the window's surface in a picture.
	pid, err := render.NewPictureId(X)
	if err != nil {
		logger.Error("new_picture_id", nil, err)
		return
	}
	render.CreatePicture(X, pid, xproto.Drawable(wid), pformat, 0, []uint32{})

	// setup.BitmapFormatScanline{Pad,Unit} and setup.BitmapFormatBitOrder
	// don't interest us here since we're only using Z format pixmaps.
	for _, pf := range setup.PixmapFormats {
		if pf.Depth == depth {
			if pf.BitsPerPixel != 32 || pf.ScanlinePad != 32 {
				logger.Error("unsuported X server", nil, errors.New("bitsperpixel or scanlinepad not supported"))
				return
			}
		}
	}

	pixid, err := xproto.NewPixmapId(X)
	if err != nil {
		logger.Error("new_pixmap_id", nil, err)
		return
	}
	_ = xproto.CreatePixmap(X, depth, pixid, xproto.Drawable(screen.Root),
		uint16(img.Bounds().Dx()), uint16(img.Bounds().Dy()))

	var bgraFormat render.Pictformat
	for _, pf := range pformats.Formats {
		if pf.Depth == depth && pf.Direct == format.format {
			bgraFormat = pf.Id
			break
		}
	}
	if bgraFormat == 0 {
		logger.Error("pictformat_not_found", nil, nil)
		return
	}

	// We could also look for the inverse pictformat.
	var encoding binary.ByteOrder
	if setup.ImageByteOrder == xproto.ImageOrderMSBFirst {
		encoding = binary.BigEndian
	} else {
		encoding = binary.LittleEndian
	}

	pixpicid, err := render.NewPictureId(X)
	if err != nil {
		logger.Error("new_picture_id", nil, err)
		return
	}
	render.CreatePicture(X, pixpicid, xproto.Drawable(pixid), bgraFormat,
		0, []uint32{})

	// Do we really need this? :/
	cid, err := xproto.NewGcontextId(X)
	if err != nil {
		logger.Error("new_gcontext_id", nil, err)
		return
	}
	_ = xproto.CreateGC(X, cid, xproto.Drawable(pixid),
		xproto.GcGraphicsExposures, []uint32{0})

	bounds := img.Bounds()
	Lstart := time.Now()

	if err := shm.Init(X); err != nil {
		logger.Error("init_mit_shm", nil, err)
		return
		// We're being lazy and resolve the 1<<16 limit of requests by sending
		// a row at a time. The encoding is also done inefficiently.
		// Also see xgbutil/xgraphics/xsurface.go.
		row := make([]byte, bounds.Dx()*4)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				encoding.PutUint32(row[x*4:], format.transform(img.At(x, y)))
			}
			_ = xproto.PutImage(X, xproto.ImageFormatZPixmap,
				xproto.Drawable(pixid), cid, uint16(bounds.Dx()), 1,
				0, int16(y),
				0, depth, row)
		}
	} else {
		rep, err := shm.QueryVersion(X).Reply()
		if err != nil {
			logger.Error("query_version", nil, err)
			return
		}
		if rep.PixmapFormat != xproto.ImageFormatZPixmap ||
			!rep.SharedPixmaps {
			logger.Error("mit_shm_unfit_failed", nil, nil)
			return
		}

		shmSize := bounds.Dx() * bounds.Dy() * 4

		// As a side note, to clean up unreferenced segments (orphans):
		//  ipcs -m | awk '$6 == "0" { print $2 }' | xargs ipcrm shm
		shmID := int(C.shmget(C.IPC_PRIVATE,
			C.size_t(shmSize), C.IPC_CREAT|0777))
		if shmID == -1 {
			// TODO: We should handle this case by falling back to PutImage,
			// if only because the allocation may hit a system limit.
			logger.Error("shmget", nil, errors.New("shmget failed"))
			return
		}

		dataRaw := C.shmat(C.int(shmID), nil, 0)
		defer C.shmdt(dataRaw)
		defer C.shmctl(C.int(shmID), C.IPC_RMID, nil)

		data := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
			Data: uintptr(dataRaw), Len: shmSize, Cap: shmSize}))
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			row := data[y*bounds.Dx()*4:]
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				encoding.PutUint32(row[x*4:], format.transform(img.At(x, y)))
			}
		}

		segid, err := shm.NewSegId(X)
		if err != nil {
			logger.Error("new_segid", nil, err)
			return
		}

		// Need to have it attached on the server before we unload the segment.
		c := shm.AttachChecked(X, segid, uint32(shmID), true /* RO */)
		if err := c.Check(); err != nil {
			logger.Error("shm_attach", nil, err)
			return
		}

		_ = shm.PutImage(X, xproto.Drawable(pixid), cid,
			uint16(bounds.Dx()), uint16(bounds.Dy()), 0, 0,
			uint16(bounds.Dx()), uint16(bounds.Dy()), 0, 0,
			depth, xproto.ImageFormatZPixmap,
			0 /* SendEvent */, segid, 0 /* Offset */)
	}

	var scale float64 = 1

	go func() {
		defer close(done)
		select {
		case <-done:
			// Clean up X11 resources
			render.FreePicture(X, pid)
			render.FreePicture(X, pixpicid)
			xproto.FreePixmap(X, pixid)
			xproto.FreeGC(X, cid)
			xproto.FreeColormap(X, mid)
			xproto.DestroyWindow(X, wid)
			X.Close()
		}
	}()

	for {
		ev, xerr := X.WaitForEvent()
		if xerr != nil {
			logger.Error("x_event", nil, xerr)
			return
		}
		if ev == nil {
			return
		}

		//log.Printf("Event: %s\n", ev)
		switch e := ev.(type) {
		case xproto.UnmapNotifyEvent:
			return

		case xproto.ConfigureNotifyEvent:
			w, h := e.Width, e.Height

			scaleX := float64(bounds.Dx()) / float64(w)
			scaleY := float64(bounds.Dy()) / float64(h)

			if scaleX < scaleY {
				scale = scaleY
			} else {
				scale = scaleX
			}

			_ = render.SetPictureTransform(X, pixpicid, render.Transform{
				F64ToFixed(scale), F64ToFixed(0), F64ToFixed(0),
				F64ToFixed(0), F64ToFixed(scale), F64ToFixed(0),
				F64ToFixed(0), F64ToFixed(0), F64ToFixed(1),
			})
			_ = render.SetPictureFilter(X, pixpicid, 8, "bilinear", nil)

		case xproto.ExposeEvent:
			_ = render.Composite(X, render.PictOpSrc,
				pixpicid, render.PictureNone, pid,
				0, 0, 0, 0, 0 /* dst-x */, 0, /* dst-y */
				uint16(float64(img.Bounds().Dx())/scale),
				uint16(float64(img.Bounds().Dy())/scale))
		}
	}
}

var done = make(chan struct{})
var x11Created = false

func Disappear() {
	// 检查当前环境是否为 X11
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	if sessionType != "x11" {
		return
	}

	// 检查是否存在 DISPLAY 变量
	display := os.Getenv("DISPLAY")
	if display == "" {
		return
	}
	// 发送关闭信号
	done <- struct{}{}
}
