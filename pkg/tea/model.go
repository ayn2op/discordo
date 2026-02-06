package tea

type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View(canvas *Canvas, area Rect)
}
