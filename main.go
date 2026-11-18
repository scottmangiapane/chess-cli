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
	Theme         string
	Pieces        string
}

func parseConfig(args []string) (Config, error) {
	config := Config{
		StockfishPath: DefaultStockfishName,
		Depth:         8,
		Threads:       runtime.NumCPU(),
		HashMB:        256,
		Theme:         "classic",
		Pieces:        "unicode",
	}

	flags := flag.NewFlagSet("chess-cli", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: chess-cli [options]")
		fmt.Fprintln(os.Stderr, "Options:")
		flags.VisitAll(func(option *flag.Flag) {
			fmt.Fprintf(os.Stderr, "  -%s (default %s): %s\n", option.Name, option.DefValue, option.Usage)
		})
	}
	flags.StringVar(&config.StockfishPath, "stockfish", config.StockfishPath, "Stockfish executable path")
	flags.IntVar(&config.Depth, "depth", config.Depth, "search depth")
	flags.IntVar(&config.Threads, "threads", config.Threads, "Stockfish threads")
	flags.IntVar(&config.HashMB, "hash-mb", config.HashMB, "Stockfish hash size in MB")
	flags.BoolVar(&config.NoColor, "no-color", false, "disable terminal colors")
	flags.StringVar(&config.Theme, "theme", config.Theme, "board theme (classic|midnight|forest)")
	flags.StringVar(&config.Pieces, "pieces", config.Pieces, "piece style (unicode|letters)")

	if err := flags.Parse(args); err != nil {
		return config, err
	}
	if flags.NArg() > 0 {
		return config, fmt.Errorf("unexpected argument %q; run chess-cli -h for usage", flags.Arg(0))
	}
	if config.Depth < 1 || config.Threads < 1 || config.HashMB < 1 {
		return config, errors.New("depth, threads, and hash-mb must be positive integers")
	}
	themeName, err := normalizeEnum("theme", config.Theme, themeNames)
	if err != nil {
		return config, err
	}
	config.Theme = themeName
	pieceStyle, err := normalizePieceStyle(config.Pieces)
	if err != nil {
		return config, err
	}
	config.Pieces = pieceStyle
	return config, nil
}

func printHelp() {
	fmt.Print(`
chess-cli — play chess against Stockfish.
You play White by default; use "new black" to play Black.

MOVES
  e4, Nf3, Bxc6+, O-O, e8=Q
  Coordinates also work: e2e4, e7e8q

COMMANDS
  help               Show help
  hint               Suggest a move
  depth [N]          Show or change search depth
  undo               Undo the last turn
  save / load        Save to or load from game.pgn
  new [white|black]  Start a new game
  theme [classic|midnight|forest]
                     Change the board theme
  pieces [unicode|letters]
                     Change how pieces are displayed
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
	theme, err := LoadTheme(config.Theme)
	if err != nil {
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
	pieceStyle := config.Pieces
	themeName := config.Theme

	reader := bufio.NewScanner(os.Stdin)
	reader.Buffer(make([]byte, 4096), 1024*1024)
	playerColor := "w"

	fen := InitialFEN
	var history []Move
	result := "*"
	notice := ""

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
		RenderGame(fen, playerColor, history, depth, notice, result, pieceStyle, theme, TerminalWidth())

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
		case "theme":
			if len(fields) == 1 {
				notice = "Theme: " + themeName
				continue
			}
			if len(fields) != 2 {
				notice = "Usage: theme [classic|midnight|forest]"
				continue
			}
			themeArg := strings.ToLower(fields[1])
			newTheme, themeErr := LoadTheme(themeArg)
			if themeErr != nil {
				notice = themeErr.Error()
			} else {
				theme, themeName = newTheme, themeArg
				notice = "Theme set to " + themeName + "."
			}
			continue
		case "pieces":
			if len(fields) == 1 {
				notice = "Piece style: " + pieceStyle
				continue
			}
			if len(fields) != 2 {
				notice = "Usage: pieces [unicode|letters]"
				continue
			}
			newStyle, styleErr := normalizePieceStyle(fields[1])
			if styleErr != nil {
				notice = styleErr.Error()
				continue
			}
			pieceStyle = newStyle
			notice = "Piece style set to " + pieceStyle + "."
			continue
		case "depth":
			if len(fields) == 1 {
				notice = fmt.Sprintf("Search depth: %d.", depth)
				continue
			}
			if len(fields) != 2 {
				notice = "Usage: depth [N]"
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
