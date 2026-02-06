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
	epoch uint32
	// cont marks a trailing cell occupied by a wide grapheme written at x-1.
	// The leading cell carries the grapheme payload; continuation cells only
	// reserve screen space.
	cont bool
}

type Frame struct {
	width  int
	height int
	// epoch increments each frame; cells/rows tagged with older epochs are
	// treated as logically empty.
	epoch uint32
	cells []canvasCell
	// rows marks rows touched in the current epoch for dirty-row blitting.
	rows []uint32
	// spanStart/spanEnd track touched x-ranges (end-exclusive) per row/epoch.
	spanStart []int
	spanEnd   []int
	spanGen   []uint32
	// dirty counts how many rows were first-marked in the current epoch.
	dirty int
	// pool interns styles so cells store compact style IDs.
	pool *stylePool
}

type stylePool struct {
	// styles is an ID-indexed table so cells can store a compact uint32 styleID
	// instead of a full tcell.Style value.
	styles []tcell.Style
	// index maps canonical style values to their stable IDs.
	index map[tcell.Style]uint32
	// lastStyle/lastID provide a tiny hot cache for repeated style writes,
	// avoiding a map lookup in dense text paths.
	lastStyle tcell.Style
	lastID    uint32
	hasLast   bool
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
	// Most callers write many adjacent cells with the same style.
	if p.hasLast && style == p.lastStyle {
		return p.lastID
	}
	if id, ok := p.index[style]; ok {
		p.lastStyle, p.lastID, p.hasLast = style, id, true
		return id
	}

	// First time we've seen this style: assign the next sequential ID, append
	// it to the ID table, and cache the reverse lookup + hot-path last-style.
	id := uint32(len(p.styles))
	p.styles = append(p.styles, style)
	p.index[style] = id
	p.lastStyle, p.lastID, p.hasLast = style, id, true
	return id
}

func (p *stylePool) style(id uint32) tcell.Style {
	// Invalid IDs can appear only from corrupted state; fall back defensively.
	if int(id) < len(p.styles) {
		return p.styles[id]
	}
	return tcell.StyleDefault
}

func NewFrame(width, height int, pool *stylePool) *Frame {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	return &Frame{
		width:     width,
		height:    height,
		cells:     make([]canvasCell, width*height),
		rows:      make([]uint32, height),
		spanStart: make([]int, height),
		spanEnd:   make([]int, height),
		spanGen:   make([]uint32, height),
		pool:      pool,
	}
}

func (f *Frame) BeginFrame() {
	// Advance frame generation so old writes are logically dropped.
	// Cells not touched in this frame read back as empty via cellAt.
	f.epoch++
	f.dirty = 0
	if f.epoch != 0 {
		return
	}

	// Rare wraparound guard: clear epoch tags so no stale generation collides
	// with the wrapped value.
	for i := range f.cells {
		f.cells[i].epoch = 0
	}
	for i := range f.rows {
		f.rows[i] = 0
	}
	f.epoch = 1
}

func (f *Frame) Size() (int, int) {
	return f.width, f.height
}

func (f *Frame) Put(x int, y int, str string, style tcell.Style) (string, int) {
	return f.putWithStyleID(x, y, str, f.pool.intern(style))
}

func (f *Frame) putWithStyleID(x int, y int, str string, styleID uint32) (string, int) {
	if x < 0 || y < 0 || y >= f.height || x >= f.width || str == "" {
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
	if width > 1 && x == f.width-1 {
		grapheme = " "
		width = 1
	}

	// Before writing, clear any same-frame wide grapheme ownership that overlaps
	// the write span so we never leave orphan continuation cells behind.
	f.clearOverlappingWideCells(x, y, width)
	f.touchSpan(y, x, x+width)

	index := y*f.width + x
	f.cells[index] = canvasCell{
		text:    canonicalCellText(grapheme),
		styleID: styleID,
		epoch:   f.epoch,
	}

	for i := range width - 1 {
		cellX := x + i + 1
		if cellX >= f.width {
			break
		}
		f.cells[index+i+1] = canvasCell{
			epoch: f.epoch,
			cont:  true,
		}
	}

	return remain, width
}

func (f *Frame) PutStr(x int, y int, str string) {
	f.PutStrStyled(x, y, str, tcell.StyleDefault)
}

func (f *Frame) PutStrStyled(x int, y int, str string, style tcell.Style) {
	styleID := f.pool.intern(style)
	for str != "" && x < f.width && y >= 0 && y < f.height {
		// Fast path for ASCII-heavy text.
		if b := str[0]; b < utf8.RuneSelf {
			// ASCII writes can overwrite a previous wide glyph, so run the same
			// overlap cleanup as putWithStyleID.
			f.clearOverlappingWideCells(x, y, 1)
			f.touchSpan(y, x, x+1)
			index := y*f.width + x
			f.cells[index] = canvasCell{
				text:    asciiText[b],
				styleID: styleID,
				epoch:   f.epoch,
			}
			x++
			str = str[1:]
			continue
		}

		remain, width := f.putWithStyleID(x, y, str, styleID)
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

func (f *Frame) style(id uint32) tcell.Style {
	return f.pool.style(id)
}

func (f *Frame) cellAt(x int, y int) canvasCell {
	if x < 0 || y < 0 || x >= f.width || y >= f.height {
		return canvasCell{}
	}
	cell := f.cells[y*f.width+x]
	// Old-generation cells are treated as empty so each frame is logically
	// self-contained without a physical clear pass.
	if cell.epoch != f.epoch {
		return canvasCell{}
	}
	return cell
}

func (f *Frame) markRowDirty(y int) {
	if f.rows[y] == f.epoch {
		return
	}
	f.rows[y] = f.epoch
	f.dirty++
}

func (f *Frame) rowDirty(y int) bool {
	if y < 0 || y >= f.height {
		return false
	}
	return f.rows[y] == f.epoch
}

func (f *Frame) hasDirtyRows() bool {
	return f.dirty > 0
}

func (f *Frame) touchSpan(y int, start int, end int) {
	if y < 0 || y >= f.height {
		return
	}
	if start < 0 {
		start = 0
	}
	if end > f.width {
		end = f.width
	}
	if start >= end {
		return
	}

	f.markRowDirty(y)
	if f.spanGen[y] != f.epoch {
		f.spanGen[y] = f.epoch
		f.spanStart[y] = start
		f.spanEnd[y] = end
		return
	}
	if start < f.spanStart[y] {
		f.spanStart[y] = start
	}
	if end > f.spanEnd[y] {
		f.spanEnd[y] = end
	}
}

func (f *Frame) rowSpan(y int) (start int, end int, ok bool) {
	if y < 0 || y >= f.height {
		return 0, 0, false
	}
	if f.spanGen[y] != f.epoch {
		return 0, 0, false
	}
	return f.spanStart[y], f.spanEnd[y], true
}

func (f *Frame) resize(width int, height int) {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	if width == f.width && height == f.height {
		return
	}

	f.width = width
	f.height = height

	cells := width * height
	if cap(f.cells) < cells {
		f.cells = make([]canvasCell, cells)
	} else {
		f.cells = f.cells[:cells]
	}

	if cap(f.rows) < height {
		f.rows = make([]uint32, height)
	} else {
		f.rows = f.rows[:height]
	}
	if cap(f.spanStart) < height {
		f.spanStart = make([]int, height)
	} else {
		f.spanStart = f.spanStart[:height]
	}
	if cap(f.spanEnd) < height {
		f.spanEnd = make([]int, height)
	} else {
		f.spanEnd = f.spanEnd[:height]
	}
	if cap(f.spanGen) < height {
		f.spanGen = make([]uint32, height)
	} else {
		f.spanGen = f.spanGen[:height]
	}
}

func (f *Frame) clearOverlappingWideCells(x int, y int, width int) {
	if width <= 0 || x < 0 || y < 0 || y >= f.height || x >= f.width {
		return
	}
	end := min(x+width, f.width)
	for col := x; col < end; col++ {
		// Resolve each target cell independently because a write can begin in the
		// middle of a pre-existing wide grapheme.
		f.clearWideAt(col, y)
	}
}

func (f *Frame) clearWideAt(x int, y int) {
	cell := f.cells[y*f.width+x]
	if cell.epoch != f.epoch {
		return
	}

	if cell.cont {
		// Writing on a continuation cell invalidates the owning wide grapheme.
		lead := x - 1
		for lead >= 0 {
			candidate := f.cells[y*f.width+lead]
			if candidate.epoch != f.epoch {
				return
			}
			if !candidate.cont {
				break
			}
			lead--
		}
		if lead < 0 {
			return
		}
		f.clearWideRun(lead, y)
		return
	}

	if x+1 < f.width {
		next := f.cells[y*f.width+x+1]
		if next.epoch == f.epoch && next.cont {
			// Writing on a lead cell with continuation to the right also replaces
			// that original wide grapheme.
			f.clearWideRun(x, y)
		}
	}
}

func (f *Frame) clearWideRun(leadX int, y int) {
	if leadX < 0 || leadX >= f.width || y < 0 || y >= f.height {
		return
	}
	f.cells[y*f.width+leadX] = canvasCell{epoch: f.epoch}
	// Clear the contiguous continuation run owned by this lead cell.
	end := leadX + 1
	for x := leadX + 1; x < f.width; x++ {
		cell := f.cells[y*f.width+x]
		if cell.epoch != f.epoch || !cell.cont {
			break
		}
		f.cells[y*f.width+x] = canvasCell{epoch: f.epoch}
		end = x + 1
	}
	f.touchSpan(y, leadX, end)
}
