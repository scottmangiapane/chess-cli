package main

import (
	"fmt"
	"strings"
)

const InitialFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

type Board [8][8]rune

func ParseBoard(fen string) (Board, string, string) {
	var board Board
	fields := strings.Fields(fen)
	if len(fields) < 4 {
		return board, "w", "-"
	}
	for r, rank := range strings.Split(fields[0], "/") {
		if r >= 8 {
			break
		}
		file := 0
		for _, piece := range rank {
			if piece >= '1' && piece <= '8' {
				file += int(piece - '0')
			} else if file < 8 {
				board[r][file] = piece
				file++
			}
		}
	}
	return board, fields[1], fields[3]
}

func Square(s string) (int, int) {
	if len(s) < 2 {
		return -1, -1
	}
	file := int(s[0] - 'a')
	rank := 8 - int(s[1]-'0')
	if file < 0 || file > 7 || rank < 0 || rank > 7 {
		return -1, -1
	}
	return rank, file
}

func pieceLabel(piece rune, style string) string {
	if style == "letters" {
		return string(piece)
	}
	switch piece {
	case 'K':
		return "♔"
	case 'Q':
		return "♕"
	case 'R':
		return "♖"
	case 'B':
		return "♗"
	case 'N':
		return "♘"
	case 'P':
		return "♙"
	case 'k':
		return "♚"
	case 'q':
		return "♛"
	case 'r':
		return "♜"
	case 'b':
		return "♝"
	case 'n':
		return "♞"
	case 'p':
		return "♟"
	default:
		return " "
	}
}

func BoardLines(fen, playerColor, pieceStyle string, theme Theme, compact bool) []string {
	board, turn, _ := ParseBoard(fen)
	flipped := playerColor == "b"
	turnName := "White"
	if turn == "b" {
		turnName = "Black"
	}
	lines := []string{PaintText(turnName+" to move", theme.Accent)}

	files := "abcdefgh"
	if flipped {
		files = "hgfedcba"
	}
	var labels strings.Builder
	if compact {
		labels.WriteString("  ")
		labels.WriteString(strings.Join(strings.Split(files, ""), " "))
		labels.WriteString(" ")
	} else {
		labels.WriteString("   ")
		for _, file := range files {
			fmt.Fprintf(&labels, " %c ", file)
		}
	}
	lines = append(lines, PaintText(labels.String(), theme.Muted))

	for row := 0; row < 8; row++ {
		boardRow := row
		if flipped {
			boardRow = 7 - row
		}
		line := fmt.Sprintf("%d  ", 8-boardRow)
		if compact {
			line = fmt.Sprintf("%d ", 8-boardRow)
		}
		var cells strings.Builder
		cells.WriteString(line)
		for col := 0; col < 8; col++ {
			boardCol := col
			if flipped {
				boardCol = 7 - col
			}
			piece := board[boardRow][boardCol]
			light := (boardRow+boardCol)%2 == 0
			background := theme.DarkSquare
			if light {
				background = theme.LightSquare
			}
			foreground := theme.WhitePiece
			if piece >= 'a' && piece <= 'z' {
				foreground = theme.BlackPiece
			}
			glyph := " "
			if piece != 0 {
				glyph = pieceLabel(piece, pieceStyle)
			}
			cell := " " + glyph + " "
			if compact {
				cell = " " + glyph
			}
			cells.WriteString(Paint(cell, foreground, background))
		}
		lines = append(lines, cells.String())
	}
	return lines
}

func RenderGame(fen, playerColor string, history []Move, depth int, notice, result, pieceStyle string, theme Theme, width int) {
	compact := width < 48
	board := BoardLines(fen, playerColor, pieceStyle, theme, compact)
	panelWidth := 32
	if width < 36 {
		panelWidth = width - 2
		if panelWidth < 12 {
			panelWidth = 12
		}
	}
	panel := PanelLines(playerColor, history, depth, notice, result, theme, panelWidth)

	if width >= 68 && !compact {
		for i := 0; i < len(board) || i < len(panel); i++ {
			if i < len(board) {
				fmt.Print(board[i])
				gap := 35 - visibleLength(board[i])
				if gap < 4 {
					gap = 4
				}
				fmt.Print(strings.Repeat(" ", gap))
			} else {
				fmt.Print(strings.Repeat(" ", 35))
			}
			if i < len(panel) {
				fmt.Println(panel[i])
			} else {
				fmt.Println()
			}
		}
		return
	}

	for _, line := range board {
		fmt.Println(line)
	}
	fmt.Println()
	for _, line := range panel {
		fmt.Println(line)
	}
}

func visibleLength(text string) int {
	// Board ANSI styling uses CSI sequences; strip them before aligning the panel.
	visible := false
	length := 0
	for _, r := range text {
		if r == '\033' {
			visible = true
			continue
		}
		if visible {
			if r == 'm' {
				visible = false
			}
			continue
		}
		length++
	}
	return length
}
