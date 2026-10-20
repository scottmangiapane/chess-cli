package main

var (
	WhiteKing   = "♚"
	WhiteQueen  = "♛"
	WhiteRook   = "♜"
	WhiteBishop = "♝"
	WhiteKnight = "♞"
	WhitePawn   = "♟"
	BlackKing   = "♔"
	BlackQueen  = "♕"
	BlackRook   = "♖"
	BlackBishop = "♗"
	BlackKnight = "♘"
	BlackPawn   = "♙"
)

func EnableColors() {
	WhiteKing = Yellow(BlackKing)
	WhiteQueen = Yellow(BlackQueen)
	WhiteRook = Yellow(BlackRook)
	WhiteBishop = Yellow(BlackBishop)
	WhiteKnight = Yellow(BlackKnight)
	WhitePawn = Yellow(BlackPawn)
	BlackKing = Blue(BlackKing)
	BlackQueen = Blue(BlackQueen)
	BlackRook = Blue(BlackRook)
	BlackBishop = Blue(BlackBishop)
	BlackKnight = Blue(BlackKnight)
	BlackPawn = Blue(BlackPawn)
}
