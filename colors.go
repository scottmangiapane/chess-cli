package main

var colorsEnabled = true

func SetColorsEnabled(enabled bool) {
	colorsEnabled = enabled
}

func color(code, text string) string {
	if !colorsEnabled {
		return text
	}
	return "\033[" + code + "m" + text + "\033[0m"
}

func Yellow(text string) string { return color("33", text) }
func Blue(text string) string   { return color("34", text) }
