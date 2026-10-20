package main

var (
	ResetCode      = "\033[0m"
	BoldCode       = "\033[1m"
	FaintCode      = "\033[2m"
	ItalicCode     = "\033[3m"
	UnderlinedCode = "\033[4m"
	BlackCode      = "\033[30m"
	RedCode        = "\033[31m"
	GreenCode      = "\033[32m"
	YellowCode     = "\033[33m"
	BlueCode       = "\033[34m"
	MagentaCode    = "\033[35m"
	CyanCode       = "\033[36m"
	WhiteCode      = "\033[37m"
)

func DisableColors() {
	ResetCode = ""
	BoldCode = ""
	FaintCode = ""
	ItalicCode = ""
	UnderlinedCode = ""
	BlackCode = ""
	RedCode = ""
	GreenCode = ""
	YellowCode = ""
	BlueCode = ""
	MagentaCode = ""
	CyanCode = ""
	WhiteCode = ""
}

func Bold(text string) string       { return BoldCode + text + ResetCode }
func Faint(text string) string      { return FaintCode + text + ResetCode }
func Italic(text string) string     { return ItalicCode + text + ResetCode }
func Underlined(text string) string { return UnderlinedCode + text + ResetCode }
func Black(text string) string      { return BlackCode + text + ResetCode }
func Red(text string) string        { return RedCode + text + ResetCode }
func Green(text string) string      { return GreenCode + text + ResetCode }
func Yellow(text string) string     { return YellowCode + text + ResetCode }
func Blue(text string) string       { return BlueCode + text + ResetCode }
func Magenta(text string) string    { return MagentaCode + text + ResetCode }
func Cyan(text string) string       { return CyanCode + text + ResetCode }
func White(text string) string      { return WhiteCode + text + ResetCode }
