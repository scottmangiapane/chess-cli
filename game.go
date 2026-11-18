package main

import (
	"fmt"
	"strings"
)

type Move struct {
	UCI    string
	SAN    string
	Before string
	After  string
}

func PanelLines(playerColor string, history []Move, depth int, notice, result string, theme Theme, width int) []string {
	if width < 12 {
		width = 12
	}
	colorName := "White"
	if playerColor == "b" {
		colorName = "Black"
	}
	lines := []string{
		PaintText("YOU: "+colorName, theme.Text),
		PaintText(fmt.Sprintf("STOCKFISH · DEPTH %d", depth), theme.Muted),
		"",
		PaintText("MOVES", theme.Accent),
	}

	if len(history) == 0 {
		lines = append(lines, PaintText("No moves yet", theme.Muted))
	} else {
		first := ((len(history)+1)/2 - 4) * 2
		if first < 0 {
			first = 0
		}
		if first > 0 {
			lines = append(lines, PaintText(fmt.Sprintf("… %d earlier moves", first/2), theme.Muted))
		}
		for i := first; i < len(history); i += 2 {
			moveLine := fmt.Sprintf("%d. %s", i/2+1, history[i].SAN)
			if i+1 < len(history) {
				moveLine += "  " + history[i+1].SAN
			}
			lines = append(lines, wrapText(moveLine, width)...)
		}
	}

	if result != "*" {
		lines = append(lines, "", PaintText("RESULT: "+result, theme.Accent))
	}
	if notice != "" {
		lines = append(lines, "")
		lines = append(lines, wrapText(notice, width)...)
	}
	return lines
}

func wrapText(text string, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	line := words[0]
	for _, word := range words[1:] {
		if len(line)+1+len(word) > width {
			lines = append(lines, line)
			line = word
		} else {
			line += " " + word
		}
	}
	return append(lines, line)
}

// AddCheckSuffix adds + or # based on the resulting position.
func AddCheckSuffix(stockfish *Stockfish, fen, san, checkers string) (string, error) {
	if strings.TrimSpace(checkers) == "" || strings.TrimSpace(checkers) == "(none)" {
		return san, nil
	}
	legal, err := stockfish.LegalMoves(fen)
	if err != nil {
		return "", err
	}
	if len(legal) == 0 {
		return san + "#", nil
	}
	return san + "+", nil
}

// GameStatus returns the PGN result and a human-readable message.
// A nonempty result other than "*" means the game has ended.
func GameStatus(stockfish *Stockfish, fen string) (string, string, error) {
	legal, err := stockfish.LegalMoves(fen)
	if err != nil {
		return "*", "", err
	}
	if len(legal) > 0 {
		return "*", "", nil
	}
	_, checkers, err := stockfish.Inspect(fen)
	if err != nil {
		return "*", "", err
	}
	fields := strings.Fields(fen)
	if len(fields) < 2 {
		return "*", "", fmt.Errorf("invalid FEN: %s", fen)
	}
	inCheck := strings.TrimSpace(checkers) != "" && strings.TrimSpace(checkers) != "(none)"
	if !inCheck {
		return "1/2-1/2", "Draw by stalemate.", nil
	}
	if fields[1] == "w" {
		return "0-1", "Checkmate. Black wins.", nil
	}
	return "1-0", "Checkmate. White wins.", nil
}

func Undo(history []Move, fen, playerColor string) ([]Move, string, bool) {
	if len(history) == 0 {
		return history, fen, false
	}
	lastMoveColor := "w"
	if (len(history)-1)%2 == 1 {
		lastMoveColor = "b"
	}
	remove := 1
	if lastMoveColor != playerColor {
		if len(history) < 2 {
			return history, fen, false
		}
		previousMoveColor := "w"
		if (len(history)-2)%2 == 1 {
			previousMoveColor = "b"
		}
		if previousMoveColor != playerColor {
			return history, fen, false
		}
		remove = 2
	}
	fen = history[len(history)-remove].Before
	history = history[:len(history)-remove]
	return history, fen, true
}
