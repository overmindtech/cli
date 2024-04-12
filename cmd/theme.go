package cmd

import (
	"github.com/charmbracelet/glamour"
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

type HexPalette struct {
	Light Palette
	Dark  Palette
	Logo  LogoPalette
}

type Palette struct {
	BgBase          string
	BgBaseHover     string
	BgShade         string
	BgSub           string
	BgBorder        string
	BgBorderHover   string
	BgDivider       string
	BgMain          string
	BgMainHover     string
	BgDanger        string
	BgDangerHover   string
	BgSuccess       string
	BgSuccessHover  string
	BgContrast      string
	BgContrastHover string
	BgWarning       string
	BgWarningHover  string
	LabelControl    string
	LabelFaint      string
	LabelMuted      string
	LabelBase       string
	LabelTitle      string
	LabelLink       string
	LabelContrast   string
}

var ColorPalette = HexPalette{
	Light: Palette{
		BgBase:          "#ffffff",
		BgBaseHover:     "#ebebeb",
		BgShade:         "#fafafa",
		BgSub:           "#ffffff",
		BgBorder:        "#e3e3e3",
		BgBorderHover:   "#d4d4d4",
		BgDivider:       "#f0f0f0",
		BgMain:          "#655add",
		BgMainHover:     "#4840a0",
		BgDanger:        "#d74249",
		BgDangerHover:   "#c8373e",
		BgSuccess:       "#5bb856",
		BgSuccessHover:  "#4da848",
		BgContrast:      "#141414",
		BgContrastHover: "#2b2b2b",
		BgWarning:       "#e59c57",
		BgWarningHover:  "#d9873a",
		LabelControl:    "#ffffff",
		LabelFaint:      "#adadad",
		LabelMuted:      "#616161",
		LabelBase:       "#383838",
		LabelTitle:      "#141414",
		LabelLink:       "#4f81ee",
		LabelContrast:   "#ffffff",
	},
	Dark: Palette{
		BgBase:          "#242428",
		BgBaseHover:     "#2d2d34",
		BgShade:         "#27272b",
		BgSub:           "#1a1a1f",
		BgBorder:        "#37373f",
		BgBorderHover:   "#434351",
		BgDivider:       "#29292e",
		BgMain:          "#7a70eb",
		BgMainHover:     "#938af5",
		BgDanger:        "#be5056",
		BgDangerHover:   "#d0494f",
		BgSuccess:       "#61ac5d",
		BgSuccessHover:  "#6ac865",
		BgContrast:      "#fafafa",
		BgContrastHover: "#ffffff",
		BgWarning:       "#ca8d53",
		BgWarningHover:  "#f0a660",
		LabelControl:    "#ffffff",
		LabelFaint:      "#616161",
		LabelMuted:      "#8c8c8c",
		LabelBase:       "#bababa",
		LabelTitle:      "#ededed",
		LabelLink:       "#688ede",
		LabelContrast:   "#1e1e24",
	},
	Logo: LogoPalette{
		a: "#1badf2",
		b: "#4b6ddf",
		c: "#5f51d5",
		d: "#c640ad",
		e: "#ef4971",
		f: "#fd6e43",
	},
}

var titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPalette.Light.BgMain)).Bold(true)
var textStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPalette.Light.LabelBase))
var addedLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPalette.Light.LabelControl)).Background(lipgloss.Color(ColorPalette.Light.BgSuccess))
var deletedLineStyle = lipgloss.NewStyle().Background(lipgloss.Color(ColorPalette.Light.BgDanger)).Foreground(lipgloss.Color(ColorPalette.Light.LabelControl))
var containerStyle = lipgloss.NewStyle().PaddingLeft(2).PaddingTop(2)

func markdownToString(markdown string) string {
	themePath := "./overmind-theme.json"
	hasDarkBackground := termenv.HasDarkBackground()
	if hasDarkBackground {
		themePath = "./overmind-theme-dark.json"
	}
	r, _ := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONFile(themePath),
	)
	out, _ := r.Render(markdown)
	return out
}
