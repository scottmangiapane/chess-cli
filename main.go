package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type Config struct {
	StockfishPath string
	Depth         int
	Threads       int
	HashMB        int
	NoColor       bool
}

func positiveEnv(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(value)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return n, nil
}

func parseConfig(args []string) (Config, error) {
	config := Config{
		StockfishPath: strings.TrimSpace(os.Getenv("STOCKFISH_PATH")),
		NoColor:       os.Getenv("NO_COLOR") != "",
	}
	if config.StockfishPath == "" {
		config.StockfishPath = DefaultStockfishName
	}

	var depthEnvErr, threadsEnvErr, hashEnvErr error
	if config.Depth, depthEnvErr = positiveEnv("DEPTH", 8); depthEnvErr != nil {
		config.Depth = 8
	}
	if config.Threads, threadsEnvErr = positiveEnv("THREADS", runtime.NumCPU()); threadsEnvErr != nil {
		config.Threads = runtime.NumCPU()
	}
	if config.HashMB, hashEnvErr = positiveEnv("HASH_MB", 256); hashEnvErr != nil {
		config.HashMB = 256
	}

	flags := flag.NewFlagSet("chess-cli", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: chess-cli [options]")
		fmt.Fprintln(os.Stderr, "Options (environment variables provide defaults):")
		flags.VisitAll(func(option *flag.Flag) {
			fmt.Fprintf(os.Stderr, "  -%s (default %s): %s\n", option.Name, option.DefValue, option.Usage)
		})
	}
	flags.StringVar(&config.StockfishPath, "stockfish", config.StockfishPath, "Stockfish executable path (STOCKFISH_PATH)")
	flags.IntVar(&config.Depth, "depth", config.Depth, "search depth (DEPTH)")
	flags.IntVar(&config.Threads, "threads", config.Threads, "Stockfish threads (THREADS)")
	flags.IntVar(&config.HashMB, "hash-mb", config.HashMB, "Stockfish hash size in MB (HASH_MB)")
	flags.BoolVar(&config.NoColor, "no-color", config.NoColor, "disable terminal colors (NO_COLOR)")

	if err := flags.Parse(args); err != nil {
		return config, err
	}
	if flags.NArg() > 0 {
		return config, fmt.Errorf("unexpected argument %q; run chess-cli -h for usage", flags.Arg(0))
	}

	provided := make(map[string]bool)
	flags.Visit(func(option *flag.Flag) { provided[option.Name] = true })
	if depthEnvErr != nil && !provided["depth"] {
		return config, depthEnvErr
	}
	if threadsEnvErr != nil && !provided["threads"] {
		return config, threadsEnvErr
	}
	if hashEnvErr != nil && !provided["hash-mb"] {
		return config, hashEnvErr
	}
	if config.Depth < 1 || config.Threads < 1 || config.HashMB < 1 {
		return config, errors.New("depth, threads, and hash-mb must be positive integers")
	}
	return config, nil
}

func printHelp() {
	fmt.Print(`
MOVES
  e4, Nf3, Bxc6+, O-O, O-O-O, e8=Q
  Coordinate moves also work: e2e4, e7e8q

COMMANDS
  help               Show help
  hint               Suggest a move
  depth N            Change search depth
  undo               Undo the last turn
  save / load        Save to or load from game.pgn
  new [white|black]  Start a new game
  resign             Resign the game
  fen                Print the current FEN
  quit               Exit
`)
}

func pause(reader *bufio.Scanner) {
	fmt.Print("\nPress Enter to continue...")
	if reader.Scan() {
		// Consume the pause input.
	}
}

func choosePlayerColor(reader *bufio.Scanner) (string, bool) {
	for {
		fmt.Print("Choose your side (white/black) [white]: ")
		if !reader.Scan() {
			return "w", false
		}
		switch strings.ToLower(strings.TrimSpace(reader.Text())) {
		case "", "w", "white":
			return "w", true
		case "b", "black":
			return "b", true
		default:
			fmt.Println("Please enter 'white' or 'black'.")
		}
	}
}

func sideToMove(fen string) string {
	fields := strings.Fields(fen)
	if len(fields) >= 2 && fields[1] == "b" {
		return "b"
	}
	return "w"
}

func applyMove(stockfish *Stockfish, fen string, history []Move, uci, san string) (string, []Move, string, string, error) {
	before := fen
	after, checkers, err := stockfish.Inspect(fen, uci)
	if err != nil {
		return fen, history, "*", "", err
	}
	san, err = AddCheckSuffix(stockfish, after, san, checkers)
	if err != nil {
		return fen, history, "*", "", err
	}

	history = append(history, Move{UCI: uci, SAN: san, Before: before, After: after})
	result, notice, err := GameStatus(stockfish, after)
	return after, history, result, notice, err
}

func playStockfishTurn(stockfish *Stockfish, fen string, history []Move, depth int) (string, []Move, string, string, error) {
	legal, err := stockfish.LegalMoves(fen)
	if err != nil {
		return fen, history, "*", "", err
	}
	if len(legal) == 0 {
		result, notice, statusErr := GameStatus(stockfish, fen)
		return fen, history, result, notice, statusErr
	}

	best, err := stockfish.BestMove(fen, depth)
	if err != nil {
		return fen, history, "*", "", err
	}
	if best == "" {
		result, notice, statusErr := GameStatus(stockfish, fen)
		return fen, history, result, notice, statusErr
	}

	san := SANBase(fen, best, legal)
	fen, history, result, notice, err := applyMove(stockfish, fen, history, best, san)
	if err == nil && result == "*" {
		notice = "Stockfish played " + history[len(history)-1].SAN
	}
	return fen, history, result, notice, err
}

func main() {
	config, err := parseConfig(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	SetColorsEnabled(!config.NoColor)

	stockfish, err := StartStockfish(config.StockfishPath, config.Threads, config.HashMB)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Stockfish:", err)
		os.Exit(1)
	}
	defer stockfish.Close()

	depth := config.Depth
	threads := config.Threads
	hashMB := config.HashMB

	reader := bufio.NewScanner(os.Stdin)
	reader.Buffer(make([]byte, 4096), 1024*1024)
	playerColor, ok := choosePlayerColor(reader)
	if !ok {
		return
	}

	fen := InitialFEN
	var history []Move
	result := "*"
	notice := "You are White. Stockfish plays Black."
	if playerColor == "b" {
		notice = "You are Black. Stockfish will make the first move."
	}

	for {
		// Stockfish moves whenever it is Stockfish's turn. This handles
		// both colors uniformly, including Stockfish opening as White.
		if result == "*" && sideToMove(fen) != playerColor {
			ClearScreen()
			fmt.Println("Stockfish is thinking...")
			var stockfishNotice string
			fen, history, result, stockfishNotice, err = playStockfishTurn(stockfish, fen, history, depth)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Stockfish:", err)
				return
			}
			notice = stockfishNotice
		}

		ClearScreen()
		fmt.Println(Bold("CHESS-CLI") + "  |  SAN notation")
		playerName := "White"
		if playerColor == "b" {
			playerName = "Black"
		}
		fmt.Printf("You: %s  |  Depth %d  |  Threads %d  |  Hash %d MB\n\n",
			playerName, depth, threads, hashMB)

		RenderBoard(fen, playerColor)
		ShowMoves(history)
		if notice != "" {
			fmt.Println("\n" + notice)
		}
		if result != "*" {
			fmt.Println("\nGame result:", result)
			fmt.Println("Type 'new' to play again, 'save' to write game.pgn, or 'quit'.")
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
		case "quit":
			return
		case "help":
			printHelp()
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
			legal, err := stockfish.LegalMoves(fen)
			if err != nil {
				notice = "Stockfish error: " + err.Error()
				continue
			}
			if len(legal) == 0 {
				notice = "There are no legal moves."
				continue
			}
			best, err := stockfish.BestMove(fen, depth)
			if err != nil {
				notice = "Stockfish error: " + err.Error()
			} else if best == "" {
				notice = "No move available."
			} else {
				notice = "Suggested move: " + SANBase(fen, best, legal)
			}
			continue
		case "undo":
			var undone bool
			history, fen, undone = Undo(history, fen, playerColor)
			if undone {
				result = "*"
				notice = "Undid the last turn."
			} else {
				notice = "Nothing to undo."
			}
			continue
		case "save":
			if len(fields) != 1 {
				notice = "Usage: save (writes to " + GamePGNFile + ")"
				continue
			}
			if err := SavePGN(history, result, playerColor); err != nil {
				notice = "Save failed: " + err.Error()
			} else {
				notice = "Saved " + GamePGNFile
			}
			continue
		case "load":
			if len(fields) != 1 {
				notice = "Usage: load (reads " + GamePGNFile + ")"
				continue
			}
			loadedFEN, loadedHistory, loadedResult, loadedColor, err := LoadPGN(stockfish, playerColor)
			if err != nil {
				notice = "Load failed: " + err.Error()
			} else {
				fen, history, result, playerColor = loadedFEN, loadedHistory, loadedResult, loadedColor
				notice = fmt.Sprintf("Loaded %s (%d half-moves).", GamePGNFile, len(history))
			}
			continue
		case "resign":
			if result == "*" {
				if playerColor == "w" {
					result = "0-1"
				} else {
					result = "1-0"
				}
				notice = "You resigned."
			}
			continue
		case "new":
			if len(fields) > 2 {
				notice = "Usage: new [white|black]"
				continue
			}
			if len(fields) == 2 {
				switch strings.ToLower(fields[1]) {
				case "white", "w":
					playerColor = "w"
				case "black", "b":
					playerColor = "b"
				default:
					notice = "Choose white or black: new white / new black"
					continue
				}
			}
			fen, history, result = InitialFEN, nil, "*"
			if err := stockfish.Send("ucinewgame"); err != nil {
				notice = "Stockfish error: " + err.Error()
				continue
			}
			if err := stockfish.Ready(); err != nil {
				notice = "Stockfish error: " + err.Error()
				continue
			}
			if playerColor == "w" {
				notice = "New game. You are White."
			} else {
				notice = "New game. You are Black."
			}
			continue
		}

		if result != "*" {
			notice = "The game has ended. Type 'new' to start another game."
			continue
		}
		if sideToMove(fen) != playerColor {
			notice = "It is Stockfish's turn."
			continue
		}

		legal, err := stockfish.LegalMoves(fen)
		if err != nil {
			notice = "Stockfish error: " + err.Error()
			continue
		}
		uci, san, err := ResolveMove(fen, input, legal)
		if err != nil {
			notice = err.Error()
			continue
		}

		fen, history, result, notice, err = applyMove(stockfish, fen, history, uci, san)
		if err != nil {
			notice = "Stockfish error: " + err.Error()
		}
	}
}
