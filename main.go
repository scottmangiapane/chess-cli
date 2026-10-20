package main

import (
	"fmt"
	"os"
)

var (
	moves    = ""
	position = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
)

func main() {
	if len(os.Args) > 1 {
		printUsage()
	}
	EnableColors() // TODO control this via a flag
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage:", os.Args[0])
}

func printBoard() {
	board := position[0:63]
	player := string(position[65])
	move := string(position[67])

	fmt.Printf("    a   b   c   d   e   f   g   h")
	fmt.Printf("  ┌───┬───┬───┬───┬───┬───┬───┬───┐")

	for _, row := range board {
	}
	board := position[0:63]
	player := string(position[65])
	move := string(position[67])

	for i := 0; i < 8; i++ {
		if i > 0 {
			fmt.Printf("  ├───┼───┼───┼───┼───┼───┼───┼───┤\n")
		}
		line := "│"
		for j := 0; j < 8; j++ {
			item := string(board[i*8+j])
			switch item {
			case "1":
				line += "   │"
			case "2":
				line += "   │   │"
			case "3":
				line += "   │   │   │"
			case "4":
				line += "   │   │   │   │"
			case "5":
				line += "   │   │   │   │   │"
			case "6":
				line += "   │   │   │   │   │   │"
			case "7":
				line += "   │   │   │   │   │   │   │"
			}
		}
	}

	fmt.Printf("  └───┴───┴───┴───┴───┴───┴───┴───┘")
	fmt.Printf("    a   b   c   d   e   f   g   h")
}
