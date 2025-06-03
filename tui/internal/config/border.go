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
	switch v.(string) {
	case "double":
		*p = borderPresetDouble()
	case "thick":
		*p = borderPresetThick()
	case "round":
		*p = borderPresetRound()
	case "light":
		*p = borderPresetLight()
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

func borderPresetLight() BorderPreset {
	return BorderPreset{
		Horizontal:  tview.BoxDrawingsLightHorizontal,
		Vertical:    tview.BoxDrawingsLightVertical,
		TopLeft:     tview.BoxDrawingsLightDownAndRight,
		TopRight:    tview.BoxDrawingsLightDownAndLeft,
		BottomLeft:  tview.BoxDrawingsLightUpAndRight,
		BottomRight: tview.BoxDrawingsLightUpAndLeft,
	}
}
