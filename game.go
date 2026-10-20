package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Move struct {
	UCI    string
	SAN    string
	Before string
	After  string
}

func ShowMoves(history []Move) {
	fmt.Println("\nMoves")

	if len(history) == 0 {
		fmt.Println("  (none yet)")
		return
	}

	for i := 0; i < len(history); i += 2 {
		fmt.Printf("%3d. %-8s", i/2+1, history[i].SAN)
		if i+1 < len(history) {
			fmt.Printf(" %s", history[i+1].SAN)
		}
		fmt.Println()
	}
}

// AddCheckSuffix adds + or # based on the resulting position.
func AddCheckSuffix(e *Engine, fen, san string) (string, error) {
	_, checkers, err := e.Inspect(fen)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(checkers) == "" ||
		strings.TrimSpace(checkers) == "(none)" {
		return san, nil
	}

	legal, err := e.LegalMoves(fen)
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
func GameStatus(e *Engine, fen string) (string, string, error) {
	legal, err := e.LegalMoves(fen)
	if err != nil {
		return "*", "", err
	}

	if len(legal) > 0 {
		return "*", "", nil
	}

	_, checkers, err := e.Inspect(fen)
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

func SavePGN(history []Move, result, filename string) error {
	if result == "" {
		result = "*"
	}

	if !strings.HasSuffix(strings.ToLower(filename), ".pgn") {
		filename += ".pgn"
	}

	var out strings.Builder

	fmt.Fprintln(&out, `[Event "Stockfish Terminal Chess"]`)
	fmt.Fprintln(&out, `[Site "Local"]`)
	fmt.Fprintf(&out, "[Date \"%s\"]\n", time.Now().Format("2006.01.02"))
	fmt.Fprintln(&out, `[Round "1"]`)
	fmt.Fprintln(&out, `[White "Player"]`)
	fmt.Fprintln(&out, `[Black "Stockfish"]`)
	fmt.Fprintf(&out, "[Result \"%s\"]\n\n", result)

	for i, move := range history {
		if i%2 == 0 {
			fmt.Fprintf(&out, "%d. ", i/2+1)
		}
		out.WriteString(move.SAN)
		out.WriteByte(' ')
	}

	out.WriteString(result)
	out.WriteByte('\n')

	return os.WriteFile(filename, []byte(out.String()), 0644)
}

func Undo(history []Move, fen string) ([]Move, string, bool) {
	if len(history) == 0 {
		return history, fen, false
	}

	// Normally remove the human move and Stockfish's reply.
	// If the game ended immediately after the human move, remove one.
	n := 2
	if len(history)%2 == 1 {
		n = 1
	}
	if n > len(history) {
		n = len(history)
	}

	fen = history[len(history)-n].Before
	history = history[:len(history)-n]
	return history, fen, true
}
