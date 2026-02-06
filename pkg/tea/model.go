package tea

type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View(frame *Frame, area Rect)
}
