# chess-cli

A small terminal chess client for playing against [Stockfish](https://stockfishchess.org/).

## Requirements

Go 1.19+ and Stockfish on `PATH`.

## Build and run

```bash
go build -o chess-cli .
./chess-cli
```

You play White by default. Use `new black` to play Black. Run `./chess-cli -h` for startup options. Options are configured with command-line flags only; use a shell alias for preferred defaults.

| Flag | Default | Description |
|---|---:|---|
| `-stockfish PATH` | `stockfish` | Stockfish executable |
| `-depth N` | `8` | Search depth |
| `-threads N` | CPU count | Stockfish threads |
| `-hash-mb N` | `256` | Stockfish hash size in MB |
| `-no-color` | off | Disable terminal colors |
| `-theme NAME` | `classic` | `classic`, `midnight`, or `forest` |
| `-pieces STYLE` | `unicode` | `unicode` or `letters` |

Example: `./chess-cli -theme midnight -pieces letters -depth 12`.

## Play

Enter moves in chess notation (`e4`, `Nf3`, `O-O`) or coordinate notation (`e2e4`). Commands: `help`, `fen`, `hint`, `depth [N]`, `theme [NAME]`, `pieces [unicode|letters]`, `undo`, `save`, `load`, `resign`, `new [white|black]`, and `quit`. `save` writes `game.pgn`; `load` reads it from the current directory.

Built-in themes color the chessboard only; the rest of the terminal keeps its configured background. Pieces can be shown as Unicode chess glyphs or letters.
