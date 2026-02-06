package tea

import (
	"fmt"
	"image"
	"os"
	"runtime"
	"runtime/pprof"

	tcell "github.com/gdamore/tcell/v3"
)

type Program struct {
	model       Model
	screen      tcell.Screen
	front       *Canvas
	back        *Canvas
	forceRedraw bool
}

func NewProgram(model Model) *Program {
	return &Program{model: model}
}

func (p *Program) Run() error {
	if p.screen == nil {
		screen, err := tcell.NewScreen()
		if err != nil {
			return err
		}
		p.screen = screen
	}

	if err := p.screen.Init(); err != nil {
		return err
	}
	defer p.screen.Fini()

	msgs := make(chan Msg)

	done := make(chan struct{})
	defer close(done)

	go func() {
		events := p.screen.EventQ()
		for {
			select {
			case <-done:
				return
			case ev, ok := <-events:
				if !ok {
					return
				}
				msg := eventToMsg(ev)
				select {
				case msgs <- msg:
				case <-done:
					return
				}
			}
		}
	}()

	p.enqueueCmd(done, msgs, p.model.Init())
	p.render()
	if path := os.Getenv("DISCORDO_MEMPROFILE"); path != "" {
		if err := writeHeapProfile(path); err != nil {
			return err
		}
	}

	var cmd Cmd
	for {
		select {
		case <-done:
			return nil
		case msg := <-msgs:
			if _, ok := msg.(quitMsg); ok {
				return nil
			}
			if batch, ok := msg.(batchMsg); ok {
				for _, c := range batch {
					p.enqueueCmd(done, msgs, c)
				}
				continue
			}
			p.model, cmd = p.model.Update(msg)
			p.enqueueCmd(done, msgs, cmd)
			p.render()
		}
	}
}

func (p *Program) render() {
	p.ensureCanvases()
	// Start a fresh logical frame on the back buffer.
	p.back.BeginFrame()
	width, height := p.back.Size()
	p.model.View(p.back, image.Rect(0, 0, width, height))
	p.blit()
	// Swap buffers so the freshly rendered frame becomes the diff baseline.
	p.front, p.back = p.back, p.front
}

func (p *Program) ensureCanvases() {
	width, height := p.screen.Size()
	if p.front != nil && p.back != nil {
		if fw, fh := p.front.Size(); fw == width && fh == height {
			return
		}
	}

	pool := newStylePool()
	p.front = NewCanvas(width, height, pool)
	p.back = NewCanvas(width, height, pool)
	p.front.BeginFrame()
	p.back.BeginFrame()
	p.forceRedraw = true
}

func (p *Program) blit() {
	width, height := p.back.Size()
	for y := range height {
		for x := range width {
			next := p.back.cellAt(x, y)
			prev := p.front.cellAt(x, y)

			// Fast path: unchanged cell in non-forced redraw mode.
			if !p.forceRedraw && cellsEqual(prev, next) {
				continue
			}

			// Continuation cells belong to a wide grapheme started at x-1.
			// Writing the lead cell already paints the full glyph width.
			if next.cont {
				continue
			}

			if next.text == "" {
				// Explicitly clear cells that were previously painted but not
				// written in this frame.
				p.screen.Put(x, y, " ", tcell.StyleDefault)
				continue
			}
			p.screen.Put(x, y, next.text, p.back.style(next.styleID))
		}
	}
	p.forceRedraw = false
	p.screen.Show()
}

func cellsEqual(a, b canvasCell) bool {
	if a.cont != b.cont {
		return false
	}
	if a.text != b.text {
		return false
	}
	return a.styleID == b.styleID
}

func (p *Program) enqueueCmd(done <-chan struct{}, msgs chan<- Msg, cmd Cmd) {
	if cmd == nil {
		return
	}
	select {
	case <-done:
		return
	default:
	}

	// Long-running commands must not block the update/render loop.
	go func(c Cmd) {
		msg := c()
		if msg == nil {
			return
		}
		select {
		case msgs <- msg:
		case <-done:
		}
	}(cmd)
}

func (p *Program) Screen() tcell.Screen {
	return p.screen
}

func (p *Program) SetScreen(screen tcell.Screen) {
	p.screen = screen
}

func eventToMsg(event tcell.Event) Msg {
	switch e := event.(type) {
	case *tcell.EventKey:
		return KeyMsg(*e)
	case *tcell.EventMouse:
		return MouseMsg(*e)
	case *tcell.EventResize:
		return ResizeMsg(*e)
	case *tcell.EventPaste:
		return PasteMsg(*e)
	case *tcell.EventFocus:
		return FocusMsg(*e)
	default:
		return e
	}
}

func writeHeapProfile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create heap profile: %w", err)
	}
	defer file.Close()

	// Force a collection so the snapshot represents live heap after startup
	// render instead of transient allocations.
	runtime.GC()
	if err := pprof.WriteHeapProfile(file); err != nil {
		return fmt.Errorf("write heap profile: %w", err)
	}
	return nil
}
