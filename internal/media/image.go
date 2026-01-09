package media

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color/palette"
	"image/png"
	"os"
	"sync"

	"github.com/BourgeoisBear/rasterm"
	"github.com/gdamore/tcell/v3"
	"golang.org/x/image/draw"
)

const (
	defaultCellWidth  = 10
	defaultCellHeight = 20
	reservedIDBase    = 0x100000
	maxCachedImages   = 100
)

func GetCellSize() (int, int) {
	w, h, ok := getCellSizeFromTerminal()
	if ok {
		return w, h
	}
	return defaultCellWidth, defaultCellHeight
}

type Protocol int

const (
	ProtoAuto Protocol = iota
	ProtoKitty
	ProtoIterm
	ProtoSixel
	ProtoAnsi
	ProtoFallback
)

func (p Protocol) String() string {
	switch p {
	case ProtoKitty:
		return "Kitty"
	case ProtoIterm:
		return "iTerm2"
	case ProtoSixel:
		return "Sixel"
	case ProtoAnsi:
		return "ANSI"
	case ProtoFallback:
		return "Fallback"
	default:
		return "Auto"
	}
}

var (
	protocolMu      sync.RWMutex
	currentProtocol Protocol
	protocolOnce    sync.Once
	forcedProtocol  *Protocol
)

func SetProtocol(p Protocol) {
	protocolMu.Lock()
	defer protocolMu.Unlock()
	forcedProtocol = &p
}

func DetectProtocol() Protocol {
	protocolMu.RLock()
	if forcedProtocol != nil {
		p := *forcedProtocol
		protocolMu.RUnlock()
		return p
	}
	protocolMu.RUnlock()

	protocolOnce.Do(func() {
		if rasterm.IsKittyCapable() {
			currentProtocol = ProtoKitty
			return
		}

		if rasterm.IsItermCapable() {
			currentProtocol = ProtoIterm
			return
		}

		term := os.Getenv("TERM")
		if term == "foot" || term == "foot-extra" {
			currentProtocol = ProtoSixel
			return
		}

		if capable, err := rasterm.IsSixelCapable(); err == nil && capable {
			currentProtocol = ProtoSixel
			return
		}

		currentProtocol = ProtoAnsi
	})
	return currentProtocol
}

func DrawImage(imgInfo *ImageInfo, x, y, w, h int, proto Protocol, clipRect image.Rectangle) {
	img := imgInfo.Image
	os.Stdout.Write([]byte("\x1b7"))
	defer os.Stdout.Write([]byte("\x1b8"))

	srcX, srcY := 0, 0
	srcW, srcH := img.Bounds().Dx(), img.Bounds().Dy()

	destX, destY := x, y
	destW, destH := w, h

	if destY < clipRect.Min.Y {
		diff := clipRect.Min.Y - destY
		if diff >= destH {
			return
		}

		ratioY := float64(srcH) / float64(h)
		pixelCrop := int(float64(diff) * ratioY)

		srcY += pixelCrop
		srcH -= pixelCrop

		destY += diff
		destH -= diff
	}

	if destY+destH > clipRect.Max.Y {
		overflow := (destY + destH) - clipRect.Max.Y
		if overflow >= destH {
			return
		}

		ratioY := float64(srcH) / float64(destH)

		pixelCrop := int(float64(overflow) * ratioY)
		srcH -= pixelCrop
		destH -= overflow
	}

	if destX < clipRect.Min.X {

		diff := clipRect.Min.X - destX
		if diff >= destW {
			return
		}

		ratioX := float64(srcW) / float64(w)
		pixelCrop := int(float64(diff) * ratioX)

		srcX += pixelCrop
		srcW -= pixelCrop

		destX += diff
		destW -= diff
	}

	if destX+destW > clipRect.Max.X {
		overflow := (destX + destW) - clipRect.Max.X
		if overflow >= destW {
			return
		}

		ratioX := float64(srcW) / float64(destW)
		pixelCrop := int(float64(overflow) * ratioX)
		srcW -= pixelCrop
		destW -= overflow
	}

	if srcW <= 0 || srcH <= 0 || destW <= 0 || destH <= 0 {
		return
	}

	termY := destY + 1
	termX := destX + 1

	moveCursor := fmt.Sprintf("\x1b[%d;%dH", termY, termX)
	os.Stdout.Write([]byte(moveCursor))

	switch proto {
	case ProtoKitty:
		action := "p"
		payload := ""

		if !imgInfo.Uploaded {
			action = "t"

			var buf bytes.Buffer
			if err := png.Encode(&buf, img); err == nil {
				data := base64.StdEncoding.EncodeToString(buf.Bytes())
				payload = data
				imgInfo.Uploaded = true
			}
		}

		id := imgInfo.KittyID

		header := fmt.Sprintf("a=%s,i=%d,q=2,c=%d,r=%d,x=%d,y=%d,w=%d,h=%d",
			action, id, destW, destH, srcX, srcY, srcW, srcH)

		if payload != "" {
			const chunkSize = 4096
			for i := 0; i < len(payload); i += chunkSize {
				end := i + chunkSize
				if end > len(payload) {
					end = len(payload)
				}

				chunk := payload[i:end]
				mVal := 1
				if end == len(payload) {
					mVal = 0
				}

				prefix := ""
				if i == 0 {
					prefix = header + ",f=100,"
				}

				cmd := fmt.Sprintf("\x1b_G%sm=%d;%s\x1b\\", prefix, mVal, chunk)
				os.Stdout.Write([]byte(cmd))
			}
		} else {
			cmd := fmt.Sprintf("\x1b_G%s;\x1b\\", header)
			os.Stdout.Write([]byte(cmd))
		}

	case ProtoIterm:

		sub, ok := img.(interface {
			SubImage(r image.Rectangle) image.Image
		})
		if ok {
			rect := image.Rect(srcX, srcY, srcX+srcW, srcY+srcH)
			origBounds := img.Bounds()
			rect = rect.Add(origBounds.Min)
			rect = rect.Intersect(origBounds)

			if !rect.Empty() {
				img = sub.SubImage(rect)
			}
		}

		opts := rasterm.ItermImgOpts{
			Width:         fmt.Sprintf("%d", destW),
			Height:        fmt.Sprintf("%d", destH),
			DisplayInline: true,
		}
		rasterm.ItermWriteImageWithOptions(os.Stdout, img, opts)

	case ProtoSixel:
		sub, ok := img.(interface {
			SubImage(r image.Rectangle) image.Image
		})
		if ok {
			rect := image.Rect(srcX, srcY, srcX+srcW, srcY+srcH)
			origBounds := img.Bounds()
			rect = rect.Add(origBounds.Min)
			rect = rect.Intersect(origBounds)
			if !rect.Empty() {
				img = sub.SubImage(rect)
			}
		}

		var paletted *image.Paletted
		if imgInfo.Paletted != nil {
			fullPaletted := imgInfo.Paletted
			subPaletted, ok := fullPaletted.SubImage(image.Rect(srcX, srcY, srcX+srcW, srcY+srcH).Add(fullPaletted.Bounds().Min)).(*image.Paletted)
			if ok {
				paletted = subPaletted
			}
		}

		if paletted == nil {
			paletted = convertToPaletted(img)
		}

		rasterm.SixelWriteImage(os.Stdout, paletted)
	case ProtoAnsi:
		fullW, fullH := imgInfo.Width, imgInfo.Height*2

		if imgInfo.SmallImage == nil || imgInfo.SmallImage.Bounds().Dx() != fullW || imgInfo.SmallImage.Bounds().Dy() != fullH {
			dst := image.NewRGBA(image.Rect(0, 0, fullW, fullH))
			draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
			imgInfo.SmallImage = dst
		}

		smallImg := imgInfo.SmallImage
		buf := bytes.Buffer{}

		origBounds := img.Bounds()
		startX := 0
		startY := 0
		if origBounds.Dx() > 0 {
			startX = srcX * fullW / origBounds.Dx()
		}
		if origBounds.Dy() > 0 {
			startY = srcY * fullH / origBounds.Dy()
		}

		for r := 0; r < h; r++ {
			termY := destY + r + 1
			termX := destX + 1
			buf.WriteString(fmt.Sprintf("\x1b[%d;%dH", termY, termX))

			yTop := startY + r*2
			yBot := yTop + 1

			for c := 0; c < w; c++ {
				x := startX + c

				if x >= fullW || yTop >= fullH {
					buf.WriteString(" ")
					continue
				}

				r1, g1, b1, _ := smallImg.At(x, yTop).RGBA()

				var r2, g2, b2 uint32
				if yBot < fullH {
					r2, g2, b2, _ = smallImg.At(x, yBot).RGBA()
				} else {
					r2, g2, b2 = 0, 0, 0
				}

				c1r, c1g, c1b := uint8(r1>>8), uint8(g1>>8), uint8(b1>>8)
				c2r, c2g, c2b := uint8(r2>>8), uint8(g2>>8), uint8(b2>>8)

				buf.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀\x1b[0m",
					c1r, c1g, c1b, c2r, c2g, c2b))
			}
		}
		os.Stdout.Write(buf.Bytes())
	}
}

func convertToPaletted(img image.Image) *image.Paletted {
	bounds := img.Bounds()
	paletted := image.NewPaletted(bounds, palette.Plan9)
	draw.FloydSteinberg.Draw(paletted, bounds, img, image.Point{})
	return paletted
}

type ImageInfo struct {
	URL        string
	Image      image.Image
	Paletted   *image.Paletted
	SmallImage image.Image
	Width      int
	Height     int
	KittyID    uint32
	Uploaded   bool
}

type ImageManager struct {
	mu       sync.RWMutex
	images   map[uint32]*ImageInfo
	urlToID  map[string]uint32
	idOrder  []uint32
	nextID   uint32
	maxItems int
}

var GlobalImageManager = NewImageManager(maxCachedImages)

func NewImageManager(maxItems int) *ImageManager {
	return &ImageManager{
		images:   make(map[uint32]*ImageInfo),
		urlToID:  make(map[string]uint32),
		idOrder:  make([]uint32, 0),
		nextID:   reservedIDBase,
		maxItems: maxItems,
	}
}

func (m *ImageManager) Register(url string, width, height int) uint32 {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existingID, ok := m.urlToID[url]; ok {
		return existingID
	}

	m.evictOldestLocked()

	id := m.nextID
	m.nextID++

	m.images[id] = &ImageInfo{
		URL:    url,
		Width:  width,
		Height: height,
	}
	m.urlToID[url] = id
	m.idOrder = append(m.idOrder, id)

	return id
}

func (m *ImageManager) evictOldestLocked() {
	for len(m.idOrder) >= m.maxItems {
		oldestID := m.idOrder[0]
		m.idOrder = m.idOrder[1:]

		if info, ok := m.images[oldestID]; ok {
			delete(m.urlToID, info.URL)
			delete(m.images, oldestID)
		}
	}
}

func (m *ImageManager) Get(id uint32) (*ImageInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	info, ok := m.images[id]
	return info, ok
}

func (m *ImageManager) SetImage(id uint32, img image.Image) {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.images[id]
	if !ok {
		return
	}

	info.Image = img
	info.KittyID = id

	if DetectProtocol() == ProtoSixel {
		info.Paletted = convertToPaletted(img)
	}
}

func (m *ImageManager) ClearAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.images = make(map[uint32]*ImageInfo)
	m.urlToID = make(map[string]uint32)
	m.idOrder = make([]uint32, 0)

	if DetectProtocol() == ProtoKitty {
		os.Stdout.Write([]byte("\x1b_Ga=d,d=a\x1b\\"))
	}
}

func (m *ImageManager) ClearScreen(screen tcell.Screen) {
	w, h := screen.Size()
	screen.LockRegion(0, 0, w, h, false)

	if DetectProtocol() == ProtoKitty {
		os.Stdout.Write([]byte("\x1b_Ga=d,d=a\x1b\\"))
	}
}

func (m *ImageManager) ScanAndDraw(screen tcell.Screen, x, y, w, h int) {
	proto := DetectProtocol()
	if proto == ProtoFallback {
		return
	}

	screenW, screenH := screen.Size()
	clipRect := image.Rect(x, 0, x+w, screenH)

	m.mu.RLock()
	defer m.mu.RUnlock()

	visited := make(map[uint32]bool)

	for row := y; row < y+h; row++ {
		if row < 0 || row >= screenH {
			continue
		}

		for col := x; col < x+w; col++ {
			if col < 0 || col >= screenW {
				continue
			}

			mainStr, style, _ := screen.Get(col, row)
			if mainStr != " " {
				continue
			}

			fg := style.GetForeground()
			hex := uint32(fg.Hex())
			if hex < reservedIDBase {
				continue
			}

			if info, exists := m.images[hex]; exists {
				if visited[hex] {
					continue
				}
				visited[hex] = true

				if info.Image != nil {
					DrawImage(info, col, row, info.Width, info.Height, proto, clipRect)
					screen.LockRegion(col, row, info.Width, info.Height, true)
				}
			}
		}
	}
}
