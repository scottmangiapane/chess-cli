package main

var (
	WhiteKing   = "♔"
	WhiteQueen  = "♕"
	WhiteRook   = "♖"
	WhiteBishop = "♗"
	WhiteKnight = "♘"
	WhitePawn   = "♙"

	BlackKing   = "♚"
	BlackQueen  = "♛"
	BlackRook   = "♜"
	BlackBishop = "♝"
	BlackKnight = "♞"
	BlackPawn   = "♟"
)

func EnableColors() {
	WhiteKing = Yellow("♔")
	WhiteQueen = Yellow("♕")
	WhiteRook = Yellow("♖")
	WhiteBishop = Yellow("♗")
	WhiteKnight = Yellow("♘")
	WhitePawn = Yellow("♙")

	BlackKing = Blue("♚")
	BlackQueen = Blue("♛")
	BlackRook = Blue("♜")
	BlackBishop = Blue("♝")
	BlackKnight = Blue("♞")
	BlackPawn = Blue("♟")
}
