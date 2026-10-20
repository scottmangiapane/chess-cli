package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func envInt(name string, fallback int) (int, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(value)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return n, nil
}

func printHelp() {
	fmt.Println(`
MOVES
  e4, Nf3, Bxc6+, O-O, O-O-O, e8=Q
  UCI coordinates also work: e2e4, e7e8q

COMMANDS
  help             Show help
  board            Redraw the board
  moves            Show move history
  fen              Print current FEN
  hint             Ask Stockfish for a suggested move
  depth N          Change search depth
  undo             Undo the last turn
  save [file.pgn]  Export the game
  resign           Resign as White
  new              Start a new game
  quit             Exit
`)
}

func pause(reader *bufio.Scanner) {
	fmt.Print("\nPress Enter to continue...")
	if reader.Scan() {
		// Consume the pause input.
	}
}

func main() {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		DisableColors()
	} else {
		EnableColors()
	}

	enginePath := os.Getenv("ENGINE")
	if enginePath == "" {
		enginePath = DefaultEnginePath
	}

	depth, err := envInt("DEPTH", 8)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	threads, err := envInt("THREADS", runtime.NumCPU())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	hashMB, err := envInt("HASH_MB", 256)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	engine, err := StartEngine(enginePath, threads, hashMB)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Stockfish:", err)
		os.Exit(1)
	}
	defer engine.Close()

	reader := bufio.NewScanner(os.Stdin)
	fen := InitialFEN
	var history []Move
	result := "*"
	notice := "Welcome! You are White. Stockfish plays Black."

	for {
		ClearScreen()

		fmt.Println(Bold("STOCKFISH CHESS") + "  |  SAN notation")
		fmt.Printf("Depth %d  |  Threads %d  |  Hash %d MB\n\n",
			depth, threads, hashMB)

		RenderBoard(fen)
		ShowMoves(history)

		if notice != "" {
			fmt.Println("\n" + notice)
		}

		if result != "*" {
			fmt.Println("\nGame result:", result)
			fmt.Println("Type 'new' to play again, 'save' to export PGN, or 'quit'.")
		} else {
			fmt.Println("\nEnter a move or type help.")
		}

		fmt.Print("\n> ")
		if !reader.Scan() {
			return
		}

		input := strings.TrimSpace(reader.Text())
		if input == "" {
			notice = ""
			continue
		}

		fields := strings.Fields(input)
		command := strings.ToLower(fields[0])
		notice = ""

		switch command {
		case "quit", "exit", "q":
			return

		case "help", "?":
			printHelp()
			pause(reader)
			continue

		case "board":
			continue

		case "moves":
			ShowMoves(history)
			pause(reader)
			continue

		case "fen":
			notice = fen
			continue

		case "depth":
			if len(fields) != 2 {
				notice = "Usage: depth 12"
				continue
			}
			n, err := strconv.Atoi(fields[1])
			if err != nil || n < 1 {
				notice = "Depth must be a positive integer."
			} else {
				depth = n
				notice = fmt.Sprintf("Depth set to %d.", depth)
			}
			continue

		case "hint":
			if result != "*" {
				notice = "The game has ended. Start a new game to continue."
				continue
			}

			legal, err := engine.LegalMoves(fen)
			if err != nil {
				notice = "Engine error: " + err.Error()
				continue
			}
			if len(legal) == 0 {
				notice = "There are no legal moves."
				continue
			}

			best, err := engine.BestMove(fen, depth)
			if err != nil {
				notice = "Engine error: " + err.Error()
			} else if best == "" {
				notice = "No move available."
			} else {
				notice = "Suggested move: " + SANBase(fen, best, legal)
			}
			continue

		case "undo", "u":
			if result == "*" || len(history) > 0 {
				var ok bool
				history, fen, ok = Undo(history, fen)
				if ok {
					result = "*"
					notice = "Undid the last turn."
				} else {
					notice = "Nothing to undo."
				}
			}
			continue

		case "save":
			filename := "game.pgn"
			if len(fields) > 1 {
				filename = fields[1]
			}
			if err := SavePGN(history, result, filename); err != nil {
				notice = "Save failed: " + err.Error()
			} else {
				notice = "Saved " + filename
			}
			continue

		case "resign":
			if result == "*" {
				result = "0-1"
				notice = "You resigned."
			}
			continue

		case "new":
			fen = InitialFEN
			history = nil
			result = "*"

			if err := engine.Send("ucinewgame"); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			if err := engine.Ready(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}

			notice = "New game. You are White."
			continue
		}

		if result != "*" {
			notice = "The game has ended. Type 'new' to start another game."
			continue
		}

		// Validate the player's move using Stockfish's legal move list.
		legal, err := engine.LegalMoves(fen)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		uci, san, err := ResolveMove(fen, input, legal)
		if err != nil {
			notice = err.Error()
			continue
		}

		before := fen
		after, _, err := engine.Inspect(fen, uci)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		san, err = AddCheckSuffix(engine, after, san)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		fen = after
		history = append(history, Move{
			UCI: uci, SAN: san, Before: before, After: after,
		})

		result, notice, err = GameStatus(engine, fen)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		if result != "*" {
			continue
		}

		// Stockfish replies.
		ClearScreen()
		fmt.Println("Stockfish is thinking...")

		legal, err = engine.LegalMoves(fen)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		best, err := engine.BestMove(fen, depth)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		if best == "" {
			result, notice, err = GameStatus(engine, fen)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			continue
		}

		blackSAN := SANBase(fen, best, legal)
		before = fen

		after, _, err = engine.Inspect(fen, best)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		blackSAN, err = AddCheckSuffix(engine, after, blackSAN)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		fen = after
		history = append(history, Move{
			UCI: best, SAN: blackSAN, Before: before, After: after,
		})

		result, notice, err = GameStatus(engine, fen)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		if result == "*" {
			notice = "Stockfish played " + blackSAN
		}
	}
}
