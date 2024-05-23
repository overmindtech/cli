package cmd

import (
	_ "embed"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

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

	if termenv.HasDarkBackground() {
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

var ScrollingDotsSpinner = spinner.Spinner{
	Frames: []string{"∙∙∙∙∙∙∙", "●∙∙∙∙∙∙", "∙●∙∙∙∙∙", "∙∙●∙∙∙∙", "∙∙∙●∙∙∙", "∙∙∙∙●∙∙", "∙∙∙∙∙●∙", "∙∙∙∙∙∙●"},
	FPS:    time.Second / 7, //nolint:gomnd
}

var ScrollingBarSpinner = spinner.Spinner{
	Frames: []string{"[    ]", "[=   ]", "[==  ]", "[=== ]", "[ ===]", "[  ==]", "[   =]", "[    ]"},
	FPS:    time.Second / 8, //nolint:gomnd
}

var DotsSpinner = spinner.Spinner{
	Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	FPS:    80 * time.Millisecond,
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

func markdownToString(markdown string) string {
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(MarkdownStyle()),
	)
	if err != nil {
		panic(fmt.Errorf("failed to initialize terminal renderer: %w", err))
	}
	out, err := r.Render(markdown)
	if err != nil {
		panic(fmt.Errorf("failed to render markdown: %w", err))
	}
	return out
}
