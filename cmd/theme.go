package cmd

import (
	_ "embed"
	"fmt"
	"image/color"
	"os"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	lipgloss "github.com/charmbracelet/lipgloss/v2"
)

// constrain the maximum terminal width to avoid readability issues with too
// long lines
const MAX_TERMINAL_WIDTH = 120

type LogoPalette struct {
	a string
	b string
	c string
	d string
	e string
	f string
}

type Palette struct {
	BgBase          color.Color
	BgBaseHover     color.Color
	BgShade         color.Color
	BgSub           color.Color
	BgBorder        color.Color
	BgBorderHover   color.Color
	BgDivider       color.Color
	BgMain          color.Color
	BgMainHover     color.Color
	BgDanger        color.Color
	BgDangerHover   color.Color
	BgSuccess       color.Color
	BgSuccessHover  color.Color
	BgContrast      color.Color
	BgContrastHover color.Color
	BgWarning       color.Color
	BgWarningHover  color.Color
	LabelControl    color.Color
	LabelFaint      color.Color
	LabelMuted      color.Color
	LabelBase       color.Color
	LabelTitle      color.Color
	LabelLink       color.Color
	LabelContrast   color.Color
}

// This is the gradient that is used in the Overmind logo
var LogoGradient = LogoPalette{
	a: "#1badf2",
	b: "#4b6ddf",
	c: "#5f51d5",
	d: "#c640ad",
	e: "#ef4971",
	f: "#fd6e43",
}

var ColorPalette Palette

func InitPalette() {
	hasDarkBG := lipgloss.HasDarkBackground(os.Stdin, os.Stderr)
	lightDark := lipgloss.LightDark(hasDarkBG)

	ColorPalette = Palette{
		BgBase:          lightDark(lipgloss.Color("#ffffff"), lipgloss.Color("#242428")),
		BgBaseHover:     lightDark(lipgloss.Color("#ebebeb"), lipgloss.Color("#2d2d34")),
		BgShade:         lightDark(lipgloss.Color("#fafafa"), lipgloss.Color("#27272b")),
		BgSub:           lightDark(lipgloss.Color("#ffffff"), lipgloss.Color("#1a1a1f")),
		BgBorder:        lightDark(lipgloss.Color("#e3e3e3"), lipgloss.Color("#37373f")),
		BgBorderHover:   lightDark(lipgloss.Color("#d4d4d4"), lipgloss.Color("#434351")),
		BgDivider:       lightDark(lipgloss.Color("#f0f0f0"), lipgloss.Color("#29292e")),
		BgMain:          lightDark(lipgloss.Color("#655add"), lipgloss.Color("#7a70eb")),
		BgMainHover:     lightDark(lipgloss.Color("#4840a0"), lipgloss.Color("#938af5")),
		BgDanger:        lightDark(lipgloss.Color("#d74249"), lipgloss.Color("#be5056")),
		BgDangerHover:   lightDark(lipgloss.Color("#c8373e"), lipgloss.Color("#d0494f")),
		BgSuccess:       lightDark(lipgloss.Color("#5bb856"), lipgloss.Color("#61ac5d")),
		BgSuccessHover:  lightDark(lipgloss.Color("#4da848"), lipgloss.Color("#6ac865")),
		BgContrast:      lightDark(lipgloss.Color("#141414"), lipgloss.Color("#fafafa")),
		BgContrastHover: lightDark(lipgloss.Color("#2b2b2b"), lipgloss.Color("#ffffff")),
		BgWarning:       lightDark(lipgloss.Color("#e59c57"), lipgloss.Color("#ca8d53")),
		BgWarningHover:  lightDark(lipgloss.Color("#d9873a"), lipgloss.Color("#f0a660")),
		LabelControl:    lightDark(lipgloss.Color("#ffffff"), lipgloss.Color("#ffffff")),
		LabelFaint:      lightDark(lipgloss.Color("#adadad"), lipgloss.Color("#616161")),
		LabelMuted:      lightDark(lipgloss.Color("#616161"), lipgloss.Color("#8c8c8c")),
		LabelBase:       lightDark(lipgloss.Color("#383838"), lipgloss.Color("#bababa")),
		LabelTitle:      lightDark(lipgloss.Color("#141414"), lipgloss.Color("#ededed")),
		LabelLink:       lightDark(lipgloss.Color("#4f81ee"), lipgloss.Color("#688ede")),
		LabelContrast:   lightDark(lipgloss.Color("#ffffff"), lipgloss.Color("#1e1e24")),
	}
}

func MarkdownStyle() ansi.StyleConfig {
	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockPrefix: "\n",
				BlockSuffix: "\n",
				Color:       getHex(ColorPalette.LabelBase),
			},
			Indent: ptrUint(2),
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Italic: ptrBool(true),
			},
			Indent:      ptrUint(1),
			IndentToken: ptrString("│ "),
		},
		List: ansi.StyleList{
			LevelIndent: 2,
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Bold:        ptrBool(true),
				Color:       getHex(ColorPalette.LabelTitle),
				BlockSuffix: "\n",
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BackgroundColor: getHex(ColorPalette.BgMain),
				Color:           getHex(ColorPalette.BgBase),
			},
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: getHex(ColorPalette.LabelMuted),
			},
		},
		H4: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "#### ",
			},
		},
		H5: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "##### ",
			},
		},
		H6: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "###### ",
				Bold:   ptrBool(false),
			},
		},
		Strikethrough: ansi.StylePrimitive{
			CrossedOut: ptrBool(true),
		},
		Emph: ansi.StylePrimitive{
			Italic: ptrBool(true),
		},
		Strong: ansi.StylePrimitive{
			Bold: ptrBool(true),
		},
		HorizontalRule: ansi.StylePrimitive{
			Color:  getHex(ColorPalette.LabelBase),
			Format: "\n--------\n",
		},
		Item: ansi.StylePrimitive{
			BlockPrefix: "• ",
		},
		Enumeration: ansi.StylePrimitive{
			BlockPrefix: ". ",
		},
		Task: ansi.StyleTask{
			Ticked:   "[✓] ",
			Unticked: "[ ] ",
		},
		Link: ansi.StylePrimitive{
			Color:       getHex(ColorPalette.LabelLink),
			Underline:   ptrBool(true),
			BlockPrefix: "(",
			BlockSuffix: ")",
		},
		LinkText: ansi.StylePrimitive{
			Bold: ptrBool(true),
		},
		Image: ansi.StylePrimitive{
			Color:       getHex(ColorPalette.LabelLink),
			Underline:   ptrBool(true),
			BlockPrefix: "(",
			BlockSuffix: ")",
		},
		ImageText: ansi.StylePrimitive{
			Color: getHex(ColorPalette.LabelLink),
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				Margin: ptrUint(4),
			},
			Theme: "solarized-light",
		},
		Table: ansi.StyleTable{
			CenterSeparator: ptrString("┼"),
			ColumnSeparator: ptrString("│"),
			RowSeparator:    ptrString("─"),
		},
		DefinitionDescription: ansi.StylePrimitive{
			BlockPrefix: "\n🠶 ",
		},
	}
}

func styleH1() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Background(ColorPalette.BgMain).
		Bold(true).
		PaddingLeft(2).
		PaddingRight(2)
}

// markdownToString converts the markdown string to a string containing ANSI
// formatting sequences with at most maxWidth visible characters per line. Set
// maxWidth to zero to use the underlying library's default.
func markdownToString(maxWidth int, markdown string) string {
	opts := []glamour.TermRendererOption{
		glamour.WithStyles(MarkdownStyle()),
	}
	if maxWidth > 0 {
		// reduce maxWidth by 4 to account for padding in the various styles
		if maxWidth > 4 {
			maxWidth -= 4
		}
		opts = append(opts, glamour.WithWordWrap(maxWidth))
	}
	r, err := glamour.NewTermRenderer(opts...)
	if err != nil {
		panic(fmt.Errorf("failed to initialize terminal renderer: %w", err))
	}
	out, err := r.Render(markdown)
	if err != nil {
		panic(fmt.Errorf("failed to render markdown: %w", err))
	}
	return out
}

func OkSymbol() string {
	if IsConhost() {
		return "OK"
	}
	return "✔︎"
}

func UnknownSymbol() string {
	if IsConhost() {
		return "??"
	}
	return "?"
}

func ErrSymbol() string {
	if IsConhost() {
		return "ERR"
	}
	return "✗"
}

func IndentSymbol() string {
	if IsConhost() {
		// because conhost symbols are wider, we also indent a space more
		return "    "
	}
	return "   "
}

func ptrBool(b bool) *bool {
	return &b
}
func ptrUint(u uint) *uint {
	return &u
}
func ptrString(s string) *string {
	return &s
}

func getHex(c color.Color) *string {
	r, g, b, _ := c.RGBA()
	// RGBA returns values in 0-65535, convert to 0-255
	retVal := fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8)) //nolint: gosec // overflows for displaying a color is not a security issue
	return &retVal
}
