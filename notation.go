package main

import (
	"fmt"
	"regexp"
	"strings"
)

var uciMovePattern = regexp.MustCompile(`^[a-h][1-8][a-h][1-8][qrbn]?$`)

func IsUCIMove(move string) bool {
	return uciMovePattern.MatchString(move)
}

func pieceLetter(piece rune) string {
	switch strings.ToUpper(string(piece)) {
	case "K", "Q", "R", "B", "N":
		return strings.ToUpper(string(piece))
	default:
		return ""
	}
}

// SANBase converts a legal UCI move into SAN without a check or
// checkmate suffix. The legal move list is supplied by Stockfish.
func SANBase(fen, move string, legal []string) string {
	if !IsUCIMove(move) {
		return move
	}

	board, _, ep := ParseBoard(fen)
	fr, ff := Square(move[:2])
	tr, tf := Square(move[2:4])

	if fr < 0 || tr < 0 {
		return move
	}

	piece := board[fr][ff]
	letter := pieceLetter(piece)

	if letter == "K" && fr == tr && ff == 4 && (tf == 6 || tf == 2) {
		if tf == 6 {
			return "O-O"
		}
		return "O-O-O"
	}

	capture := board[tr][tf] != 0

	// En passant captures land on an empty square.
	if strings.EqualFold(string(piece), "p") &&
		ff != tf && ep == move[2:4] {
		capture = true
	}

	var san strings.Builder

	if letter == "" {
		if capture {
			san.WriteByte(move[0])
		}
	} else {
		san.WriteString(letter)

		var competitors []string

		for _, other := range legal {
			if other == move || !IsUCIMove(other) ||
				other[2:4] != move[2:4] {
				continue
			}

			or, of := Square(other[:2])
			if or >= 0 && board[or][of] == piece {
				competitors = append(competitors, other)
			}
		}

		if len(competitors) > 0 {
			sameFile, sameRank := false, false

			for _, other := range competitors {
				or, of := Square(other[:2])
				if of == ff {
					sameFile = true
				}
				if or == fr {
					sameRank = true
				}
			}

			switch {
			case !sameFile:
				san.WriteByte(move[0])
			case !sameRank:
				san.WriteByte(move[1])
			default:
				san.WriteString(move[:2])
			}
		}
	}

	if capture {
		san.WriteByte('x')
	}

	san.WriteString(move[2:4])

	if len(move) == 5 {
		san.WriteByte('=')
		san.WriteString(strings.ToUpper(move[4:5]))
	}

	return san.String()
}

func normalizeSAN(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "0-0-0", "O-O-O")
	s = strings.ReplaceAll(s, "0-0", "O-O")

	// Check/mate suffixes and optional annotation punctuation
	// don't affect matching against the legal move.
	s = strings.TrimRight(s, "+#!?")
	return s
}

// ResolveMove accepts SAN or UCI coordinates and only returns a
// move if it matches one of Stockfish's legal moves.
func ResolveMove(fen, input string, legal []string) (string, string, error) {
	input = strings.TrimSpace(input)

	if IsUCIMove(strings.ToLower(input)) {
		input = strings.ToLower(input)

		for _, move := range legal {
			if move == input {
				return move, SANBase(fen, move, legal), nil
			}
		}
		return "", "", fmt.Errorf("illegal move: %s", input)
	}

	want := normalizeSAN(input)
	found, san := "", ""

	for _, move := range legal {
		candidate := SANBase(fen, move, legal)
		if normalizeSAN(candidate) == want {
			if found != "" {
				return "", "", fmt.Errorf("ambiguous move: %s", input)
			}
			found, san = move, candidate
		}
	}

	if found == "" {
		return "", "", fmt.Errorf("no legal move matches %q", input)
	}

	return found, san, nil
}
