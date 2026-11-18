package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type Theme struct {
	LightSquare string
	DarkSquare  string
	WhitePiece  string
	BlackPiece  string
	Text        string
	Muted       string
	Accent      string
}

var builtInThemes = map[string]Theme{
	"classic": {
		LightSquare: "#f0d9b5", DarkSquare: "#b58863",
		WhitePiece: "#fffdf4", BlackPiece: "#171717", Text: "#eeeeee", Muted: "#a3a3a3", Accent: "#e0b35a",
	},
	"midnight": {
		LightSquare: "#718096", DarkSquare: "#334155",
		WhitePiece: "#f8fafc", BlackPiece: "#111827", Text: "#e2e8f0", Muted: "#94a3b8", Accent: "#38bdf8",
	},
	"forest": {
		LightSquare: "#e4e8cf", DarkSquare: "#6b8e5a",
		WhitePiece: "#fffdf0", BlackPiece: "#102015", Text: "#e7efe1", Muted: "#a5b99f", Accent: "#c1dc91",
	},
}

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
var colorsEnabled = true

var themeNames = []string{"classic", "midnight", "forest"}
var pieceStyles = []string{"unicode", "letters"}

func normalizeEnum(kind, value string, choices []string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, choice := range choices {
		if value == choice {
			return value, nil
		}
	}
	return "", fmt.Errorf("invalid %s %q; choose one of: %s", kind, value, strings.Join(choices, ", "))
}

func normalizePieceStyle(style string) (string, error) {
	return normalizeEnum("pieces", style, pieceStyles)
}

func SetColorsEnabled(enabled bool) { colorsEnabled = enabled }

func LoadTheme(name string) (Theme, error) {
	name, err := normalizeEnum("theme", name, themeNames)
	if err != nil {
		return Theme{}, err
	}
	return builtInThemes[name], nil
}

func colorCode(hex string, foreground bool) (string, error) {
	if !hexColor.MatchString(hex) {
		return "", errors.New("invalid hex color")
	}
	r, _ := strconv.ParseInt(hex[1:3], 16, 64)
	g, _ := strconv.ParseInt(hex[3:5], 16, 64)
	b, _ := strconv.ParseInt(hex[5:7], 16, 64)
	kind := 48
	if foreground {
		kind = 38
	}
	return fmt.Sprintf("\033[%d;2;%d;%d;%dm", kind, r, g, b), nil
}

func Paint(text, foreground, background string) string {
	if !colorsEnabled {
		return text
	}
	fg, _ := colorCode(foreground, true)
	bg, _ := colorCode(background, false)
	return fg + bg + text + "\033[0m"
}

func PaintText(text, foreground string) string {
	if !colorsEnabled {
		return text
	}
	fg, _ := colorCode(foreground, true)
	return fg + text + "\033[0m"
}

func ClearScreen() {
	// Reset colors before clearing so the terminal uses its own default
	// background rather than filling the screen with a theme color.
	fmt.Print("\033[0m\033[2J\033[H")
}

func TerminalWidth() int {
	command := exec.Command("stty", "size")
	command.Stdin = os.Stdin
	if output, err := command.Output(); err == nil {
		fields := strings.Fields(string(output))
		if len(fields) == 2 {
			if width, err := strconv.Atoi(fields[1]); err == nil && width > 0 {
				return width
			}
		}
	}
	return 80
}
