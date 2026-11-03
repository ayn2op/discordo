package themepicker

// ShowMsg shows the themes with Theme selected.
type ShowMsg struct {
	Theme string
}

// PreviewMsg previews Theme.
type PreviewMsg struct {
	Theme string
}

// SelectedMsg picks Theme.
type SelectedMsg struct {
	Theme string
}

// CancelMsg goes back to the previous theme.
type CancelMsg struct{}
