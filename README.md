# chess-cli

A small terminal chess client. Play as White or Black against [Stockfish](https://stockfishchess.org/).

## Requirements

Go 1.19+ and Stockfish on `PATH` (or set `STOCKFISH_PATH`).

## Build and run

```bash
go build -o chess-cli .
./chess-cli
```

Choose a side when prompted. Run `./chess-cli -h` to see configuration options. Each option can also be set with its matching environment variable; command-line flags take precedence.

| Flag | Environment variable | Default |
|---|---|---|
| `-stockfish PATH` | `STOCKFISH_PATH` | `stockfish` on `PATH` |
| `-depth N` | `DEPTH` | `8` |
| `-threads N` | `THREADS` | Number of CPUs |
| `-hash-mb N` | `HASH_MB` | `256` |
| `-no-color` | `NO_COLOR` | Colors enabled |

For example: `./chess-cli -stockfish /path/to/stockfish -depth 12 -threads 4 -hash-mb 512`. Set `NO_COLOR=1` or use `-no-color` to disable colors.

## Play

Enter moves in SAN (`e4`, `Nf3`, `O-O`) or coordinate notation (`e2e4`). Commands: `help`, `fen`, `hint`, `depth N`, `undo`, `save`, `load`, `resign`, `new [white|black]`, and `quit`. `save` writes `game.pgn`; `load` reads it from the current directory. Loading supports standard starting positions.
