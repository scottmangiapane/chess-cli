package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

const GamePGNFile = "game.pgn"

var pgnTagPattern = regexp.MustCompile(`^\[([A-Za-z0-9_]+)\s+"((?:\\.|[^"\\])*)"\]$`)

func SavePGN(history []Move, result, playerColor string) error {
	if result == "" {
		result = "*"
	}
	if playerColor != "b" {
		playerColor = "w"
	}

	white, black := "Player", "Stockfish"
	if playerColor == "b" {
		white, black = "Stockfish", "Player"
	}

	var out strings.Builder
	fmt.Fprintln(&out, `[Event "Chess CLI"]`)
	fmt.Fprintln(&out, `[Site "Local"]`)
	fmt.Fprintf(&out, "[Date \"%s\"]\n", time.Now().Format("2006.01.02"))
	fmt.Fprintln(&out, `[Round "1"]`)
	fmt.Fprintf(&out, "[White \"%s\"]\n", white)
	fmt.Fprintf(&out, "[Black \"%s\"]\n", black)
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

	return os.WriteFile(GamePGNFile, []byte(out.String()), 0644)
}

// LoadPGN loads a standard-starting-position game by replaying each SAN move
// through Stockfish. It returns a complete new game state only on success.
func LoadPGN(stockfish *Stockfish, defaultPlayerColor string) (string, []Move, string, string, error) {
	file, err := os.Open(GamePGNFile)
	if err != nil {
		return "", nil, "", defaultPlayerColor, fmt.Errorf("open %s: %w", GamePGNFile, err)
	}
	defer file.Close()

	var movetext strings.Builder
	playerColor := defaultPlayerColor
	declaredResult := ""
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimPrefix(strings.TrimSpace(scanner.Text()), "\uFEFF")
		if matches := pgnTagPattern.FindStringSubmatch(line); matches != nil {
			key, value := strings.ToLower(matches[1]), unescapePGNTag(matches[2])
			switch key {
			case "white":
				if strings.EqualFold(value, "Player") {
					playerColor = "w"
				}
			case "black":
				if strings.EqualFold(value, "Player") {
					playerColor = "b"
				}
			case "result":
				declaredResult = value
			case "fen":
				return "", nil, "", playerColor, fmt.Errorf("%s uses a custom starting position; only standard chess games are supported", GamePGNFile)
			case "setup":
				if value == "1" {
					return "", nil, "", playerColor, fmt.Errorf("%s uses a custom starting position; only standard chess games are supported", GamePGNFile)
				}
			}
			continue
		}
		movetext.WriteString(line)
		movetext.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return "", nil, "", playerColor, fmt.Errorf("read %s: %w", GamePGNFile, err)
	}

	tokens, err := tokenizePGNMovetext(movetext.String())
	if err != nil {
		return "", nil, "", playerColor, fmt.Errorf("parse %s: %w", GamePGNFile, err)
	}

	fen := InitialFEN
	var history []Move
	result := declaredResult
	ply := 0
	for _, token := range tokens {
		if isPGNResult(token) {
			result = token
			break
		}
		if token == "" || strings.HasPrefix(token, "$") || strings.EqualFold(token, "e.p.") {
			continue
		}

		legal, err := stockfish.LegalMoves(fen)
		if err != nil {
			return "", nil, "", playerColor, fmt.Errorf("read legal moves at ply %d: %w", ply+1, err)
		}
		uci, san, err := ResolveMove(fen, token, legal)
		if err != nil {
			return "", nil, "", playerColor, fmt.Errorf("invalid PGN move %q at ply %d: %w", token, ply+1, err)
		}
		after, checkers, err := stockfish.Inspect(fen, uci)
		if err != nil {
			return "", nil, "", playerColor, fmt.Errorf("apply PGN move %q at ply %d: %w", token, ply+1, err)
		}
		san, err = AddCheckSuffix(stockfish, after, san, checkers)
		if err != nil {
			return "", nil, "", playerColor, fmt.Errorf("inspect PGN move %q at ply %d: %w", token, ply+1, err)
		}
		history = append(history, Move{UCI: uci, SAN: san, Before: fen, After: after})
		fen = after
		ply++
	}

	if result == "" || result == "*" {
		result, _, err = GameStatus(stockfish, fen)
		if err != nil {
			return "", nil, "", playerColor, fmt.Errorf("check loaded game status: %w", err)
		}
	}
	if !isPGNResult(result) {
		return "", nil, "", playerColor, fmt.Errorf("invalid PGN result %q", result)
	}

	return fen, history, result, playerColor, nil
}

func unescapePGNTag(value string) string {
	value = strings.ReplaceAll(value, `\"`, `"`)
	return strings.ReplaceAll(value, `\\`, `\`)
}

func isPGNResult(token string) bool {
	switch token {
	case "1-0", "0-1", "1/2-1/2", "*":
		return true
	default:
		return false
	}
}

// tokenizePGNMovetext removes comments and variations, then separates move
// numbers from SAN even when they are attached (for example, "1.e4").
func tokenizePGNMovetext(input string) ([]string, error) {
	input = strings.TrimPrefix(input, "\uFEFF")
	var clean strings.Builder
	braceDepth, variationDepth := 0, 0
	inLineComment := false

	for _, r := range input {
		switch {
		case inLineComment:
			if r == '\n' {
				inLineComment = false
				clean.WriteByte(' ')
			}
		case braceDepth > 0:
			if r == '{' {
				braceDepth++
			} else if r == '}' {
				braceDepth--
				if braceDepth == 0 {
					clean.WriteByte(' ')
				}
			}
		case variationDepth > 0:
			if r == '(' {
				variationDepth++
			} else if r == ')' {
				variationDepth--
				if variationDepth == 0 {
					clean.WriteByte(' ')
				}
			}
		case r == ';':
			inLineComment = true
		case r == '{':
			braceDepth = 1
			clean.WriteByte(' ')
		case r == '(':
			variationDepth = 1
			clean.WriteByte(' ')
		case r == ')':
			return nil, fmt.Errorf("unexpected closing variation parenthesis")
		default:
			clean.WriteRune(r)
		}
	}
	if braceDepth != 0 {
		return nil, fmt.Errorf("unterminated brace comment")
	}
	if variationDepth != 0 {
		return nil, fmt.Errorf("unterminated variation")
	}

	var tokens []string
	for _, raw := range strings.Fields(clean.String()) {
		token := strings.TrimSpace(raw)
		if token == "" {
			continue
		}
		token = stripPGNMoveNumber(token)
		if token == "" {
			continue
		}
		// Ignore standalone annotation glyphs; attached glyphs are removed by
		// ResolveMove's SAN normalization.
		if token == "!" || token == "?" || token == "!!" || token == "??" || token == "!?" || token == "?!" {
			continue
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func stripPGNMoveNumber(token string) string {
	i := 0
	for i < len(token) && token[i] >= '0' && token[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(token) || token[i] != '.' {
		return token
	}
	for i < len(token) && token[i] == '.' {
		i++
	}
	return token[i:]
}
