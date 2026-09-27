package qr

import (
	"strings"
	"testing"

	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	"github.com/skip2/go-qrcode"
)

func TestModelView(t *testing.T) {
	t.Run("draws the code above the status", func(t *testing.T) {
		const width, height = 80, 24
		screen, err := tcell.NewTerminfoScreenFromTty(vt.NewMockTerm(vt.MockOptSize{X: width, Y: height}))
		if err != nil {
			t.Fatal(err)
		}
		if err := screen.Init(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(screen.Fini)

		m := NewModel()
		code, err := qrcode.New("test", qrcode.Low)
		if err != nil {
			t.Fatal(err)
		}
		m.qrCode = code
		m.setStatus("status")
		m.View().Draw(screen, tview.Rectangle{Width: width, Height: height})

		var drawn strings.Builder
		for y := range height {
			for x := range width {
				str, _, _ := screen.Get(x, y)
				drawn.WriteString(str)
			}
			drawn.WriteByte('\n')
		}
		if !strings.Contains(drawn.String(), "status") || !strings.ContainsAny(drawn.String(), "█▀▄") {
			t.Fatalf("missing content:\n%s", drawn.String())
		}
	})
}
