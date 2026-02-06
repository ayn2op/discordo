package tea

import (
	"strings"
	"unicode/utf8"

	tcell "github.com/gdamore/tcell/v3"
	"github.com/rivo/uniseg"
)

var asciiText = func() [utf8.RuneSelf]string {
	var out [utf8.RuneSelf]string
	for i := range utf8.RuneSelf {
		out[i] = string(rune(i))
	}
	return out
}()

type canvasCell struct {
	text    string
	styleID uint32
	// epoch tags cells written in the current frame. Cells from older frames are
	// treated as empty without needing an explicit full-canvas clear.
	epoch uint16
	// cont marks a trailing cell occupied by a wide grapheme written at x-1.
	// The leading cell carries the grapheme payload; continuation cells only
	// reserve screen space.
	cont bool
}

type Canvas struct {
	width  int
	height int
	epoch  uint16
	cells  []canvasCell
	pool   *stylePool
}

type stylePool struct {
	styles []tcell.Style
	index  map[tcell.Style]uint32
}

func newStylePool() *stylePool {
	p := &stylePool{
		styles: []tcell.Style{tcell.StyleDefault},
		index:  make(map[tcell.Style]uint32, 8),
	}
	p.index[tcell.StyleDefault] = 0
	return p
}

func (p *stylePool) intern(style tcell.Style) uint32 {
	if id, ok := p.index[style]; ok {
		return id
	}

	id := uint32(len(p.styles))
	p.styles = append(p.styles, style)
	p.index[style] = id
	return id
}

func (p *stylePool) style(id uint32) tcell.Style {
	if int(id) < len(p.styles) {
		return p.styles[id]
	}
	return tcell.StyleDefault
}

func NewCanvas(width, height int, pool *stylePool) *Canvas {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	return &Canvas{
		width:  width,
		height: height,
		cells:  make([]canvasCell, width*height),
		pool:   pool,
	}
}

func (c *Canvas) BeginFrame() {
	// Advance frame generation so old writes are logically dropped.
	// Cells not touched in this frame read back as empty via cellAt.
	c.epoch++
	if c.epoch != 0 {
		return
	}

	// Rare wraparound guard: clear epoch tags so no stale generation collides
	// with the wrapped value.
	for i := range c.cells {
		c.cells[i].epoch = 0
	}
	c.epoch = 1
}

func (c *Canvas) Size() (int, int) {
	return c.width, c.height
}

func (c *Canvas) Put(x int, y int, str string, style tcell.Style) (string, int) {
	return c.putWithStyleID(x, y, str, c.pool.intern(style))
}

func (c *Canvas) putWithStyleID(x int, y int, str string, styleID uint32) (string, int) {
	if x < 0 || y < 0 || y >= c.height || x >= c.width || str == "" {
		return str, 0
	}

	state := -1
	grapheme, remain, width, _ := uniseg.FirstGraphemeClusterInString(str, state)
	if grapheme == "" || width <= 0 {
		return remain, 0
	}

	// Keep writes in-bounds. If a wide grapheme starts at the last column,
	// degrade to a single-cell blank, matching terminal behavior for clipped
	// wide glyphs.
	if width > 1 && x == c.width-1 {
		grapheme = " "
		width = 1
	}

	index := y*c.width + x
	c.cells[index] = canvasCell{
		text:    canonicalCellText(grapheme),
		styleID: styleID,
		epoch:   c.epoch,
	}

	for i := range width - 1 {
		cellX := x + i + 1
		if cellX >= c.width {
			break
		}
		c.cells[index+i+1] = canvasCell{
			epoch: c.epoch,
			cont:  true,
		}
	}

	return remain, width
}

func (c *Canvas) PutStr(x int, y int, str string) {
	c.PutStrStyled(x, y, str, tcell.StyleDefault)
}

func (c *Canvas) PutStrStyled(x int, y int, str string, style tcell.Style) {
	styleID := c.pool.intern(style)
	for str != "" && x < c.width && y >= 0 && y < c.height {
		// Fast path for ASCII-heavy UI text.
		if b := str[0]; b < utf8.RuneSelf {
			index := y*c.width + x
			c.cells[index] = canvasCell{
				text:    asciiText[b],
				styleID: styleID,
				epoch:   c.epoch,
			}
			x++
			str = str[1:]
			continue
		}

		remain, width := c.putWithStyleID(x, y, str, styleID)
		if width <= 0 || remain == str {
			return
		}
		str = remain
		x += width
	}
}

func canonicalCellText(s string) string {
	if len(s) == 1 && s[0] < utf8.RuneSelf {
		return asciiText[s[0]]
	}

	// Grapheme slices can alias large source strings; clone so cells don't
	// retain unrelated backing buffers across frames.
	return strings.Clone(s)
}

func (c *Canvas) style(id uint32) tcell.Style {
	return c.pool.style(id)
}

func (c *Canvas) cellAt(x int, y int) canvasCell {
	if x < 0 || y < 0 || x >= c.width || y >= c.height {
		return canvasCell{}
	}
	cell := c.cells[y*c.width+x]
	// Old-generation cells are treated as empty so each frame is logically
	// self-contained without a physical clear pass.
	if cell.epoch != c.epoch {
		return canvasCell{}
	}
	return cell
}
