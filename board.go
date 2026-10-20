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
		return WhiteKing
	case 'Q':
		return WhiteQueen
	case 'R':
		return WhiteRook
	case 'B':
		return WhiteBishop
	case 'N':
		return WhiteKnight
	case 'P':
		return WhitePawn
	case 'k':
		return BlackKing
	case 'q':
		return BlackQueen
	case 'r':
		return BlackRook
	case 'b':
		return BlackBishop
	case 'n':
		return BlackKnight
	case 'p':
		return BlackPawn
	default:
		return " "
	}
}

func RenderBoard(fen string) {
	board, turn, _ := ParseBoard(fen)

	fields := strings.Fields(fen)
	moveNumber := "1"
	if len(fields) >= 6 {
		moveNumber = fields[5]
	}

	fmt.Println("    a   b   c   d   e   f   g   h")
	fmt.Println("  ┌───┬───┬───┬───┬───┬───┬───┬───┐")

	for r := 0; r < 8; r++ {
		if r > 0 {
			fmt.Println("  ├───┼───┼───┼───┼───┼───┼───┼───┤")
		}

		fmt.Printf("%d │", 8-r)
		for f := 0; f < 8; f++ {
			fmt.Printf(" %s │", PieceGlyph(board[r][f]))
		}
		fmt.Printf(" %d", 8-r)

		switch r {
		case 3:
			if turn == "w" {
				fmt.Print("   White to move")
			} else {
				fmt.Print("   Black to move")
			}
		case 4:
			fmt.Printf("   Move #%s", moveNumber)
		}
		fmt.Println()
	}

	fmt.Println("  └───┴───┴───┴───┴───┴───┴───┴───┘")
	fmt.Println("    a   b   c   d   e   f   g   h")
}

func ClearScreen() {
	fmt.Print("\033[2J\033[H")
}
