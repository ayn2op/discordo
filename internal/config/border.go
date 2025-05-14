package config

import "github.com/rivo/tview"

type BorderPreset struct {
	Horizontal  rune
	Vertical    rune
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune
}

func (p *BorderPreset) UnmarshalTOML(v any) error {
	// Handle str and rune values
	switch val := v.(type) {
	case string:
		switch val {
		case "double":
			*p = borderPresetDouble()
		case "thick":
			*p = borderPresetThick()
		case "round":
			*p = borderPresetRound()
		case "hidden":
			*p = BorderPreset{
				Horizontal:  ' ',
				Vertical:    ' ',
				TopLeft:     ' ',
				TopRight:    ' ',
				BottomLeft:  ' ',
				BottomRight: ' ',
			}
		}
	case map[string]any:
		var bp BorderPreset
		
		if h, ok := val["Horizontal"].(int64); ok {
			bp.Horizontal = rune(h)
		}
		if v, ok := val["Vertical"].(int64); ok {
			bp.Vertical = rune(v)
		}
		if tl, ok := val["TopLeft"].(int64); ok {
			bp.TopLeft = rune(tl)
		}
		if tr, ok := val["TopRight"].(int64); ok {
			bp.TopRight = rune(tr)
		}
		if bl, ok := val["BottomLeft"].(int64); ok {
			bp.BottomLeft = rune(bl)
		}
		if br, ok := val["BottomRight"].(int64); ok {
			bp.BottomRight = rune(br)
		}
		
		*p = bp
	}

	return nil
}

func borderPresetDouble() BorderPreset {
	return BorderPreset{
		Horizontal:  tview.BoxDrawingsDoubleHorizontal,
		Vertical:    tview.BoxDrawingsDoubleVertical,
		TopLeft:     tview.BoxDrawingsDoubleDownAndRight,
		TopRight:    tview.BoxDrawingsDoubleDownAndLeft,
		BottomLeft:  tview.BoxDrawingsDoubleUpAndRight,
		BottomRight: tview.BoxDrawingsDoubleUpAndLeft,
	}
}

func borderPresetThick() BorderPreset {
	return BorderPreset{
		Horizontal:  tview.BoxDrawingsHeavyHorizontal,
		Vertical:    tview.BoxDrawingsHeavyVertical,
		TopLeft:     tview.BoxDrawingsHeavyDownAndRight,
		TopRight:    tview.BoxDrawingsHeavyDownAndLeft,
		BottomLeft:  tview.BoxDrawingsHeavyUpAndRight,
		BottomRight: tview.BoxDrawingsHeavyUpAndLeft,
	}
}

func borderPresetRound() BorderPreset {
	return BorderPreset{
		Horizontal:  tview.BoxDrawingsLightHorizontal,
		Vertical:    tview.BoxDrawingsLightVertical,
		TopLeft:     tview.BoxDrawingsLightArcDownAndRight,
		TopRight:    tview.BoxDrawingsLightArcDownAndLeft,
		BottomLeft:  tview.BoxDrawingsLightArcUpAndRight,
		BottomRight: tview.BoxDrawingsLightArcUpAndLeft,
	}
}
