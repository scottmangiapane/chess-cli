
# Stockfish Client

A dependency-free Go terminal client for [Stockfish](https://stockfishchess.org).

## Requirements

- Go 1.19 or later
- Stockfish installed locally

The default engine path is `/opt/homebrew/bin/stockfish` (Homebrew on Apple Silicon macOS).

## Build

```sh
go build -o chess-cli .
./chess-cli
```

Override the engine path or search settings with environment variables:

```sh
ENGINE=/path/to/stockfish DEPTH=12 THREADS=4 HASH_MB=512 ./chess-cli
```

Disable terminal colors with `NO_COLOR=1`.

## Moves

Enter standard algebraic notation (SAN):

- `e4` — pawn move
- `Nf3` — knight move
- `Bxc6+` — capture giving check
- `O-O` / `O-O-O` — castling
- `e8=Q` — promotion

UCI coordinate moves such as `e2e4` are also accepted.

## Commands

- `help` — show commands
- `board` — redraw the board
- `moves` — display move history
- `fen` — display the current FEN
- `hint` — ask Stockfish for a move suggestion
- `depth N` — change search depth
- `undo` — take back the last turn
- `save [file.pgn]` — export the game
- `resign` — resign
- `new` — start a new game
- `quit` — exit

## Implementation

The Go client uses only the standard library. Stockfish handles legal move
generation and engine search through UCI. The client formats and matches SAN,
renders the board, maintains move history, and exports PGN.

