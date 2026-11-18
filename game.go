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

func ShowMoves(history []Move) {
	if len(history) == 0 {
		return
	}

	// Keep the live display compact; the complete game remains in history
	// and is included when saving the PGN.
	first := ((len(history)+1)/2 - 4) * 2
	if first < 0 {
		first = 0
	}
	if first > 0 {
		earlier := first / 2
		label := "moves"
		if earlier == 1 {
			label = "move"
		}
		fmt.Printf("… %d earlier %s\n", earlier, label)
	}

	for i := first; i < len(history); i += 2 {
		fmt.Printf("%d. %s", i/2+1, history[i].SAN)
		if i+1 < len(history) {
			fmt.Printf(" %s", history[i+1].SAN)
		}
		fmt.Println()
	}
}

// AddCheckSuffix adds + or # based on the resulting position.
func AddCheckSuffix(stockfish *Stockfish, fen, san, checkers string) (string, error) {
	if strings.TrimSpace(checkers) == "" ||
		strings.TrimSpace(checkers) == "(none)" {
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

	inCheck := strings.TrimSpace(checkers) != "" &&
		strings.TrimSpace(checkers) != "(none)"

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

	// Moves alternate starting with White. At the player's turn, undo the
	// player's previous move and Stockfish's reply. If the game ended on
	// the player's move, undo just that move. A Black player cannot undo
	// before making a move because only Stockfish's opening move exists.
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
