package clipboard

// Format represents the type of clipboard content.
type Format int

const (
	FmtText  Format = iota // plain text
	FmtImage               // image data
)
