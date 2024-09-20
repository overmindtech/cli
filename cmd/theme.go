package cmd

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
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
	BgBase          lipgloss.AdaptiveColor
	BgBaseHover     lipgloss.AdaptiveColor
	BgShade         lipgloss.AdaptiveColor
	BgSub           lipgloss.AdaptiveColor
	BgBorder        lipgloss.AdaptiveColor
	BgBorderHover   lipgloss.AdaptiveColor
	BgDivider       lipgloss.AdaptiveColor
	BgMain          lipgloss.AdaptiveColor
	BgMainHover     lipgloss.AdaptiveColor
	BgDanger        lipgloss.AdaptiveColor
	BgDangerHover   lipgloss.AdaptiveColor
	BgSuccess       lipgloss.AdaptiveColor
	BgSuccessHover  lipgloss.AdaptiveColor
	BgContrast      lipgloss.AdaptiveColor
	BgContrastHover lipgloss.AdaptiveColor
	BgWarning       lipgloss.AdaptiveColor
	BgWarningHover  lipgloss.AdaptiveColor
	LabelControl    lipgloss.AdaptiveColor
	LabelFaint      lipgloss.AdaptiveColor
	LabelMuted      lipgloss.AdaptiveColor
	LabelBase       lipgloss.AdaptiveColor
	LabelTitle      lipgloss.AdaptiveColor
	LabelLink       lipgloss.AdaptiveColor
	LabelContrast   lipgloss.AdaptiveColor
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

var ColorPalette = Palette{
	BgBase: lipgloss.AdaptiveColor{
		Light: "#ffffff",
		Dark:  "#242428",
	},
	BgBaseHover: lipgloss.AdaptiveColor{
		Light: "#ebebeb",
		Dark:  "#2d2d34",
	},
	BgShade: lipgloss.AdaptiveColor{
		Light: "#fafafa",
		Dark:  "#27272b",
	},
	BgSub: lipgloss.AdaptiveColor{
		Light: "#ffffff",
		Dark:  "#1a1a1f",
	},
	BgBorder: lipgloss.AdaptiveColor{
		Light: "#e3e3e3",
		Dark:  "#37373f",
	},
	BgBorderHover: lipgloss.AdaptiveColor{
		Light: "#d4d4d4",
		Dark:  "#434351",
	},
	BgDivider: lipgloss.AdaptiveColor{
		Light: "#f0f0f0",
		Dark:  "#29292e",
	},
	BgMain: lipgloss.AdaptiveColor{
		Light: "#655add",
		Dark:  "#7a70eb",
	},
	BgMainHover: lipgloss.AdaptiveColor{
		Light: "#4840a0",
		Dark:  "#938af5",
	},
	BgDanger: lipgloss.AdaptiveColor{
		Light: "#d74249",
		Dark:  "#be5056",
	},
	BgDangerHover: lipgloss.AdaptiveColor{
		Light: "#c8373e",
		Dark:  "#d0494f",
	},
	BgSuccess: lipgloss.AdaptiveColor{
		Light: "#5bb856",
		Dark:  "#61ac5d",
	},
	BgSuccessHover: lipgloss.AdaptiveColor{
		Light: "#4da848",
		Dark:  "#6ac865",
	},
	BgContrast: lipgloss.AdaptiveColor{
		Light: "#141414",
		Dark:  "#fafafa",
	},
	BgContrastHover: lipgloss.AdaptiveColor{
		Light: "#2b2b2b",
		Dark:  "#ffffff",
	},
	BgWarning: lipgloss.AdaptiveColor{
		Light: "#e59c57",
		Dark:  "#ca8d53",
	},
	BgWarningHover: lipgloss.AdaptiveColor{
		Light: "#d9873a",
		Dark:  "#f0a660",
	},
	LabelControl: lipgloss.AdaptiveColor{
		Light: "#ffffff",
		Dark:  "#ffffff",
	},
	LabelFaint: lipgloss.AdaptiveColor{
		Light: "#adadad",
		Dark:  "#616161",
	},
	LabelMuted: lipgloss.AdaptiveColor{
		Light: "#616161",
		Dark:  "#8c8c8c",
	},
	LabelBase: lipgloss.AdaptiveColor{
		Light: "#383838",
		Dark:  "#bababa",
	},
	LabelTitle: lipgloss.AdaptiveColor{
		Light: "#141414",
		Dark:  "#ededed",
	},
	LabelLink: lipgloss.AdaptiveColor{
		Light: "#4f81ee",
		Dark:  "#688ede",
	},
	LabelContrast: lipgloss.AdaptiveColor{
		Light: "#ffffff",
		Dark:  "#1e1e24",
	},
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

func MarkdownStyle() ansi.StyleConfig {
	var bgBase string
	var bgMain string
	var labelBase string
	var labelLink string
	var labelMuted string
	var labelTitle string

	if lipgloss.HasDarkBackground() {
		bgBase = ColorPalette.BgBase.Dark
		bgMain = ColorPalette.BgMain.Dark
		labelBase = ColorPalette.LabelBase.Dark
		labelLink = ColorPalette.LabelLink.Dark
		labelMuted = ColorPalette.LabelMuted.Dark
		labelTitle = ColorPalette.LabelTitle.Dark
	} else {
		bgBase = ColorPalette.BgBase.Light
		bgMain = ColorPalette.BgMain.Light
		labelBase = ColorPalette.LabelBase.Light
		labelLink = ColorPalette.LabelLink.Light
		labelMuted = ColorPalette.LabelMuted.Light
		labelTitle = ColorPalette.LabelTitle.Light
	}

	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockPrefix: "\n",
				BlockSuffix: "\n",
				Color:       &labelBase,
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
				Color:       &labelTitle,
				BlockSuffix: "\n",
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BackgroundColor: &bgMain,
				Color:           &bgBase,
			},
		},
		H3: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: &labelMuted,
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
			Color:  &labelBase,
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
			Color:       &labelLink,
			Underline:   ptrBool(true),
			BlockPrefix: "(",
			BlockSuffix: ")",
		},
		LinkText: ansi.StylePrimitive{
			Bold: ptrBool(true),
		},
		Image: ansi.StylePrimitive{
			Color:       &labelLink,
			Underline:   ptrBool(true),
			BlockPrefix: "(",
			BlockSuffix: ")",
		},
		ImageText: ansi.StylePrimitive{
			Color: &labelLink,
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

var titleStyle = lipgloss.NewStyle().Foreground(ColorPalette.BgMain).Bold(true)
var textStyle = lipgloss.NewStyle().Foreground(ColorPalette.LabelBase)
var addedLineStyle = lipgloss.NewStyle().Foreground(ColorPalette.LabelControl).Background(ColorPalette.BgSuccess)
var deletedLineStyle = lipgloss.NewStyle().Background(ColorPalette.BgDanger).Foreground(ColorPalette.LabelControl)
var containerStyle = lipgloss.NewStyle().PaddingLeft(2).PaddingTop(2)

func styleH1() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Background(ColorPalette.BgMain).
		Bold(true).
		PaddingLeft(2).
		PaddingRight(2)
}

func styleH2() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(ColorPalette.BgMain).
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

// wrap ensures that the text is wrapped to the given width and everything but
// the first line is indented by the requested amount. Consider that the current
// implementation is very naive and for large indent values, the first line
// might not be wrapped too early.
//
// Indent is ignored when the requested indent is larger than the current width.
// This is expected to only occur in edge cases, e.g. when the terminal is
// resiyed to very narrow.
func wrap(s string, width, indent int) string {
	if indent > width {
		indent = 0
	}

	return strings.ReplaceAll(wordwrap.String(s, width-indent), "\n", "\n"+strings.Repeat(" ", indent))
}

func OkSymbol() string {
	if IsConhost() {
		return "OK"
	}
	return "✔︎"
}

func RenderOk() string {
	return lipgloss.NewStyle().Foreground(ColorPalette.BgSuccess).Render(OkSymbol())
}

func UnknownSymbol() string {
	if IsConhost() {
		return "??"
	}
	return "?"
}
func RenderUnknown() string {
	return lipgloss.NewStyle().Foreground(ColorPalette.BgWarning).Render(UnknownSymbol())
}

func ErrSymbol() string {
	if IsConhost() {
		return "ERR"
	}
	return "✗"
}
func RenderErr() string {
	return lipgloss.NewStyle().Foreground(ColorPalette.BgDanger).Render(ErrSymbol())
}

func IndentSymbol() string {
	if IsConhost() {
		// because conhost symbols are wider, we also indent a space more
		return "    "
	}
	return "   "
}
