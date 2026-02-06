package tea

import (
	"errors"
	"fmt"
	"image"
	"runtime/debug"
	"sync"

	tcell "github.com/gdamore/tcell/v3"
)

type Program struct {
	model   Model
	modelMu sync.Mutex

	screen tcell.Screen
	// front holds the last frame that has been flushed to the screen.
	front *Frame
	// back is the current frame being rendered before a swap.
	back *Frame
	// forceRedraw bypasses diffing and repaints every cell on the next blit.
	forceRedraw bool
}

var ErrProgramPanic = errors.New("tea program panicked")

func NewProgram(model Model) *Program {
	return &Program{model: model}
}

func (p *Program) Run() (returnErr error) {
	errs := make(chan error, 1)
	defer func() {
		if r := recover(); r != nil {
			returnErr = p.handleRunPanic("run loop", r)
		}
	}()

	// Initialize terminal resources once before any background loops start.
	if err := p.initScreen(); err != nil {
		return err
	}
	defer p.screen.Fini()

	msgs := make(chan Msg)

	// done is the shared shutdown signal for renderer/input/cmd goroutines.
	done := make(chan struct{})
	invalidate, waitForRenderLoop := p.startRenderLoop(done, errs)
	waitForInputLoop := p.startInputLoop(done, msgs, errs)
	defer func() {
		close(done)
		waitForInputLoop()
		waitForRenderLoop()
	}()

	// Kick off model init command and paint the initial frame synchronously.
	p.enqueueCmd(done, msgs, errs, p.initCmd())
	p.render()

	for {
		select {
		case err := <-errs:
			return err
		case msg := <-msgs:
			// Control messages are handled internally and do not reach Update.
			if _, ok := msg.(quitMsg); ok {
				return nil
			}
			if batch, ok := msg.(batchMsg); ok {
				for _, c := range batch {
					p.enqueueCmd(done, msgs, errs, c)
				}
				continue
			}
			if sequence, ok := msg.(sequenceMsg); ok {
				p.runSequence(done, msgs, errs, sequence)
				continue
			}

			// Regular messages go through Update, then trigger async command work
			// and a coalesced render invalidation.
			cmd := p.updateModel(msg)
			p.enqueueCmd(done, msgs, errs, cmd)
			invalidate()
		}
	}
}

func (p *Program) initScreen() error {
	if p.screen == nil {
		screen, err := tcell.NewScreen()
		if err != nil {
			return err
		}
		p.screen = screen
	}
	return p.screen.Init()
}

func (p *Program) startInputLoop(done <-chan struct{}, msgs chan<- Msg, errs chan<- error) (wait func()) {
	var wg sync.WaitGroup
	wg.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				p.handleGoroutinePanic(errs, "input loop", r)
			}
		}()
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
	})
	return wg.Wait
}

func (p *Program) startRenderLoop(done <-chan struct{}, errs chan<- error) (invalidate func(), wait func()) {
	renderSignal := make(chan struct{}, 1)
	var wg sync.WaitGroup

	wg.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				p.handleGoroutinePanic(errs, "render loop", r)
			}
		}()
		for {
			select {
			case <-done:
				return
			case <-renderSignal:
				p.render()
			}
		}
	})

	invalidate = func() {
		// Coalesce bursts of updates into a single pending render request.
		select {
		case renderSignal <- struct{}{}:
		default:
		}
	}

	return invalidate, wg.Wait
}

func (p *Program) initCmd() Cmd {
	p.modelMu.Lock()
	defer p.modelMu.Unlock()
	return p.model.Init()
}

func (p *Program) updateModel(msg Msg) Cmd {
	var cmd Cmd
	p.modelMu.Lock()
	defer p.modelMu.Unlock()
	p.model, cmd = p.model.Update(msg)
	return cmd
}

func (p *Program) render() {
	p.ensureFrames()
	// Start a fresh logical frame on the back buffer.
	p.back.BeginFrame()
	width, height := p.back.Size()
	p.modelMu.Lock()
	p.model.View(p.back, image.Rect(0, 0, width, height))
	p.modelMu.Unlock()
	p.blit()
	// Swap buffers so the freshly rendered frame becomes the diff baseline.
	p.front, p.back = p.back, p.front
}

func (p *Program) ensureFrames() {
	width, height := p.screen.Size()
	if p.front != nil && p.back != nil {
		if fw, fh := p.front.Size(); fw == width && fh == height {
			return
		}

		// Reuse backing slices on resize when capacity allows.
		p.front.resize(width, height)
		p.back.resize(width, height)
		p.front.BeginFrame()
		p.back.BeginFrame()
		p.forceRedraw = true
		return
	}

	pool := newStylePool()
	p.front = NewFrame(width, height, pool)
	p.back = NewFrame(width, height, pool)
	p.front.BeginFrame()
	p.back.BeginFrame()
	p.forceRedraw = true
}

func (p *Program) blit() {
	// Nothing changed in either generation; skip the full cell scan.
	if !p.forceRedraw && !p.back.hasDirtyRows() && !p.front.hasDirtyRows() {
		return
	}

	width, height := p.back.Size()
	changed := false
	for y := range height {
		// If neither frame touched this row, no cell in the row can differ.
		if !p.forceRedraw && !p.back.rowDirty(y) && !p.front.rowDirty(y) {
			continue
		}
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
				changed = true
				continue
			}
			p.screen.Put(x, y, next.text, p.back.style(next.styleID))
			changed = true
		}
	}
	p.forceRedraw = false
	if !changed {
		return
	}
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

func (p *Program) enqueueCmd(done <-chan struct{}, msgs chan<- Msg, errs chan<- error, cmd Cmd) {
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
		defer func() {
			if r := recover(); r != nil {
				p.handleGoroutinePanic(errs, "command", r)
			}
		}()
		msg := c()
		p.dispatchCmdMsg(done, msgs, errs, msg)
	}(cmd)
}

func (p *Program) runSequence(done <-chan struct{}, msgs chan<- Msg, errs chan<- error, sequence sequenceMsg) {
	go func(cmds sequenceMsg) {
		defer func() {
			if r := recover(); r != nil {
				p.handleGoroutinePanic(errs, "sequence", r)
			}
		}()
		for _, cmd := range cmds {
			if cmd == nil {
				continue
			}
			msg := cmd()
			p.dispatchCmdMsg(done, msgs, errs, msg)
		}
	}(sequence)
}

func (p *Program) dispatchCmdMsg(done <-chan struct{}, msgs chan<- Msg, errs chan<- error, msg Msg) {
	if msg == nil {
		return
	}
	switch m := msg.(type) {
	case batchMsg:
		for _, cmd := range m {
			p.enqueueCmd(done, msgs, errs, cmd)
		}
	case sequenceMsg:
		for _, cmd := range m {
			if cmd == nil {
				continue
			}
			next := cmd()
			p.dispatchCmdMsg(done, msgs, errs, next)
		}
	default:
		select {
		case msgs <- msg:
		case <-done:
		}
	}
}

func (p *Program) handleRunPanic(where string, recovered any) error {
	return fmt.Errorf("%w in %s: %v\n%s", ErrProgramPanic, where, recovered, debug.Stack())
}

func (p *Program) handleGoroutinePanic(errs chan<- error, where string, recovered any) {
	select {
	case errs <- fmt.Errorf("%w in %s: %v\n%s", ErrProgramPanic, where, recovered, debug.Stack()):
	default:
	}
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
