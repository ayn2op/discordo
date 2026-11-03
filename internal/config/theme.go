package config

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

var errInvalidType = errors.New("invalid type")

var (
	attributes = enum[tcell.AttrMask]{
		{"bold", tcell.AttrBold},
		{"dim", tcell.AttrDim},
		{"italic", tcell.AttrItalic},
		{"blink", tcell.AttrBlink},
		{"reverse", tcell.AttrReverse},
		{"strikethrough", tcell.AttrStrikeThrough},
	}
	underlines = enum[tcell.UnderlineStyle]{
		{"", tcell.UnderlineStyleNone},
		{"solid", tcell.UnderlineStyleSolid},
		{"double", tcell.UnderlineStyleDouble},
		{"curly", tcell.UnderlineStyleCurly},
		{"dotted", tcell.UnderlineStyleDotted},
		{"dashed", tcell.UnderlineStyleDashed},
	}
)

type Style struct{ tview.Style }

func (s *Style) UnmarshalTOML(value any) error {
	table, ok := value.(map[string]any)
	if !ok {
		return errInvalidType
	}

	// Reset on new styles
	s.Style = tcell.StyleDefault

	for key, value := range table {
		if key == "attributes" {
			s.parseAttributes(value)
			continue
		}

		name, ok := value.(string)
		if !ok {
			continue
		}

		if err := s.set(key, name); err != nil {
			return err
		}
	}

	return nil
}

// set applies the option key with the value name.
func (s *Style) set(key, name string) error {
	switch key {
	case "foreground":
		s.Style = s.Foreground(tcell.GetColor(name))
	case "background":
		s.Style = s.Background(tcell.GetColor(name))
	case "underline":
		underline, err := underlines.parse(name)
		if err != nil {
			return err
		}

		s.Style = s.Underline(underline)
	case "underline_color":
		s.Style = s.Underline(tcell.GetColor(name))
	}

	return nil
}

// parseAttributes applies one attribute name or a list of them.
func (s *Style) parseAttributes(value any) {
	switch value := value.(type) {
	case string:
		s.parseAttribute(value)
	case []any:
		for _, name := range stringValues(value) {
			s.parseAttribute(name)
		}
	}
}

func (s *Style) parseAttribute(name string) {
	if name == "underline" {
		s.Style = s.Underline(true)
		return
	}

	// An unknown attribute parses to no attribute.
	attr, _ := attributes.parse(name)
	s.Style = s.Attributes(s.GetAttributes() | attr) //nolint:staticcheck
}

// MarshalTOML writes the style as the inline table UnmarshalTOML reads.
func (s Style) MarshalTOML() ([]byte, error) {
	var fields []string
	add := func(key string, value any) {
		encoded, _ := json.Marshal(value)
		fields = append(fields, key+" = "+string(encoded))
	}

	if foreground := s.GetForeground(); foreground != tcell.ColorDefault {
		add("foreground", colorName(foreground))
	}

	if background := s.GetBackground(); background != tcell.ColorDefault {
		add("background", colorName(background))
	}

	switch names := s.attributeNames(); len(names) {
	case 0:
	case 1:
		add("attributes", names[0])
	default:
		add("attributes", names)
	}

	if underline := s.GetUnderlineStyle(); underline != tcell.UnderlineStyleNone {
		name, _ := underlines.text(underline)
		add("underline", string(name))
	}

	if underlineColor := s.GetUnderlineColor(); underlineColor != tcell.ColorDefault {
		add("underline_color", colorName(underlineColor))
	}

	if fields == nil {
		return []byte("{}"), nil
	}

	return []byte("{ " + strings.Join(fields, ", ") + " }"), nil
}

func (s Style) attributeNames() []string {
	var names []string
	for _, attr := range attributes {
		if s.GetAttributes()&attr.value != 0 {
			names = append(names, attr.name)
		}
	}

	return names
}

// colorName returns the first of the names of target, or its hex code if it has none.
func colorName(target tcell.Color) string {
	var names []string
	for name, color := range tcell.ColorNames {
		if color == target {
			names = append(names, name)
		}
	}

	if names == nil {
		return target.CSS()
	}

	return slices.Min(names)
}

type (
	ThemeConfig struct {
		Background BackgroundThemeConfig `toml:"background"`

		Primary   SwatchThemeConfig `toml:"primary"`
		Secondary SwatchThemeConfig `toml:"secondary"`
		Success   Style             `toml:"success"`
		Warning   Style             `toml:"warning"`
		Danger    Style             `toml:"danger"`
	}

	BackgroundThemeConfig struct {
		Base    Style `toml:"base"`
		Weak    Style `toml:"weak"`
		Neutral Style `toml:"neutral"`
		Strong  Style `toml:"strong"`
	}

	SwatchThemeConfig struct {
		Base   Style `toml:"base"`
		Weak   Style `toml:"weak"`
		Strong Style `toml:"strong"`
	}
)
