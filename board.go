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

	ranks := strings.Split(fields[0], "/")

	for r, rank := range ranks {
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

func PieceGlyph(piece rune) string {
	switch piece {
	case 'K':
		return Yellow("♔")
	case 'Q':
		return Yellow("♕")
	case 'R':
		return Yellow("♖")
	case 'B':
		return Yellow("♗")
	case 'N':
		return Yellow("♘")
	case 'P':
		return Yellow("♙")
	case 'k':
		return Blue("♚")
	case 'q':
		return Blue("♛")
	case 'r':
		return Blue("♜")
	case 'b':
		return Blue("♝")
	case 'n':
		return Blue("♞")
	case 'p':
		return Blue("♟")
	default:
		return " "
	}
}

func RenderBoard(fen, playerColor string) {
	board, turn, _ := ParseBoard(fen)
	fields := strings.Fields(fen)
	moveNumber := "1"
	if len(fields) >= 6 {
		moveNumber = fields[5]
	}

	flipped := playerColor == "b"
	turnName := "White"
	if turn == "b" {
		turnName = "Black"
	}
	fmt.Printf("%s. %s to move\n", moveNumber, turnName)
	files := "a b c d e f g h"
	if flipped {
		files = "h g f e d c b a"
	}
	fmt.Println("  " + files)

	for row := 0; row < 8; row++ {
		boardRow := row
		if flipped {
			boardRow = 7 - row
		}
		fmt.Print(8 - boardRow)
		for col := 0; col < 8; col++ {
			boardCol := col
			if flipped {
				boardCol = 7 - col
			}
			fmt.Printf(" %s", PieceGlyph(board[boardRow][boardCol]))
		}
		fmt.Println()
	}
}

func ClearScreen() {
	fmt.Print("\033[2J\033[H")
}
