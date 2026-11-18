package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultStockfishName = "stockfish"
	StockfishStartupWait = 10 * time.Second
	StockfishCommandWait = 10 * time.Second
)

type Stockfish struct {
	cmd   *exec.Cmd
	input io.WriteCloser
	lines chan string
}

func ResolveStockfishPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		path = DefaultStockfishName
	}

	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand Stockfish path %q: %w", path, err)
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}

	resolved, err := exec.LookPath(path)
	if err != nil {
		if path == DefaultStockfishName {
			return "", errors.New("Stockfish was not found on PATH; install it or specify its executable with -stockfish or STOCKFISH_PATH")
		}
		return "", fmt.Errorf("Stockfish executable %q was not found or is not executable: %w", path, err)
	}
	return resolved, nil
}

func StartStockfish(path string, threads, hashMB int) (*Stockfish, error) {
	resolved, err := ResolveStockfishPath(path)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(resolved)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open Stockfish input: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("open Stockfish output: %w", err)
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("start Stockfish at %q: %w", resolved, err)
	}

	s := &Stockfish{cmd: cmd, input: stdin, lines: make(chan string, 1024)}
	go func() {
		defer close(s.lines)
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 4096), 1024*1024)
		for scanner.Scan() {
			s.lines <- scanner.Text()
		}
	}()

	fail := func(err error) (*Stockfish, error) {
		s.terminate()
		if details := strings.TrimSpace(stderr.String()); details != "" {
			err = fmt.Errorf("%w (Stockfish stderr: %s)", err, details)
		}
		return nil, err
	}

	if err := s.Send("uci"); err != nil {
		return fail(fmt.Errorf("start UCI handshake: %w", err))
	}
	name, err := s.waitForUCI(StockfishStartupWait)
	if err != nil {
		return fail(fmt.Errorf("Stockfish did not complete its UCI handshake: %w", err))
	}
	if !strings.Contains(strings.ToLower(name), "stockfish") {
		return fail(fmt.Errorf("%q identified itself as %q, not Stockfish", resolved, name))
	}

	if err := s.Send(fmt.Sprintf("setoption name Threads value %d", threads)); err != nil {
		return fail(err)
	}
	if err := s.Send(fmt.Sprintf("setoption name Hash value %d", hashMB)); err != nil {
		return fail(err)
	}
	if err := s.Ready(); err != nil {
		return fail(fmt.Errorf("initialize Stockfish: %w", err))
	}
	if err := s.Send("ucinewgame"); err != nil {
		return fail(err)
	}
	if err := s.Ready(); err != nil {
		return fail(fmt.Errorf("initialize new Stockfish game: %w", err))
	}

	// This client uses Stockfish's perft and debug commands for chess rules
	// and position inspection. Verify both before allowing a game to start.
	moves, err := s.LegalMoves(InitialFEN)
	if err != nil {
		return fail(fmt.Errorf("Stockfish perft check failed: %w", err))
	}
	if len(moves) != 20 {
		return fail(fmt.Errorf("Stockfish perft check returned %d opening moves; expected 20", len(moves)))
	}
	fen, _, err := s.Inspect(InitialFEN)
	if err != nil {
		return fail(fmt.Errorf("Stockfish position-inspection check failed: %w", err))
	}
	if fen != InitialFEN {
		return fail(fmt.Errorf("Stockfish position-inspection check returned unexpected FEN %q", fen))
	}

	return s, nil
}

func (s *Stockfish) Send(command string) error {
	if s == nil || s.input == nil {
		return errors.New("Stockfish is not running")
	}
	if _, err := fmt.Fprintln(s.input, command); err != nil {
		return fmt.Errorf("write to Stockfish: %w", err)
	}
	return nil
}

func (s *Stockfish) nextLine() (string, error) {
	line, ok := <-s.lines
	if !ok {
		return "", errors.New("Stockfish closed its output")
	}
	return line, nil
}

func (s *Stockfish) waitForUCI(timeout time.Duration) (string, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var name string
	for {
		select {
		case line, ok := <-s.lines:
			if !ok {
				return "", errors.New("process exited before completing the UCI handshake")
			}
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "id name ") {
				name = strings.TrimSpace(strings.TrimPrefix(line, "id name "))
			}
			if line == "uciok" {
				if name == "" {
					return "", errors.New("UCI handshake did not include a Stockfish name")
				}
				return name, nil
			}
		case <-timer.C:
			return "", fmt.Errorf("timed out after %s waiting for uciok", timeout)
		}
	}
}

func (s *Stockfish) Ready() error {
	if err := s.Send("isready"); err != nil {
		return err
	}
	return s.waitFor("readyok", StockfishCommandWait)
}

func (s *Stockfish) waitFor(expected string, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case line, ok := <-s.lines:
			if !ok {
				return fmt.Errorf("Stockfish exited before responding with %s", expected)
			}
			if strings.TrimSpace(line) == expected {
				return nil
			}
		case <-timer.C:
			return fmt.Errorf("timed out after %s waiting for %s", timeout, expected)
		}
	}
}

func (s *Stockfish) terminate() {
	if s == nil || s.cmd == nil || s.cmd.Process == nil {
		return
	}
	if s.input != nil {
		_ = s.input.Close()
	}
	_ = s.cmd.Process.Kill()
	_ = s.cmd.Wait()
}

func (s *Stockfish) Close() {
	if s == nil || s.cmd == nil || s.cmd.Process == nil {
		return
	}
	_ = s.Send("quit")
	if s.input != nil {
		_ = s.input.Close()
	}
	_ = s.cmd.Wait()
}

func (s *Stockfish) Position(fen string, moves ...string) error {
	command := "position fen " + fen
	if len(moves) > 0 {
		command += " moves " + strings.Join(moves, " ")
	}
	return s.Send(command)
}

func (s *Stockfish) LegalMoves(fen string) ([]string, error) {
	if err := s.Position(fen); err != nil {
		return nil, err
	}
	if err := s.Send("go perft 1"); err != nil {
		return nil, err
	}

	timer := time.NewTimer(StockfishCommandWait)
	defer timer.Stop()
	var moves []string
	for {
		select {
		case line, ok := <-s.lines:
			if !ok {
				return nil, errors.New("Stockfish closed its output during perft")
			}
			if strings.HasPrefix(line, "Nodes searched:") {
				return moves, nil
			}
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			move := strings.TrimSuffix(fields[0], ":")
			if IsUCIMove(move) {
				moves = append(moves, move)
			}
		case <-timer.C:
			return nil, fmt.Errorf("timed out after %s waiting for Stockfish perft output", StockfishCommandWait)
		}
	}
}

func (s *Stockfish) Inspect(fen string, moves ...string) (string, string, error) {
	if err := s.Position(fen, moves...); err != nil {
		return "", "", err
	}
	if err := s.Send("d"); err != nil {
		return "", "", err
	}
	if err := s.Send("isready"); err != nil {
		return "", "", err
	}

	timer := time.NewTimer(StockfishCommandWait)
	defer timer.Stop()
	var resultFEN, checkers string
	var sawFEN, sawCheckers bool
	for {
		select {
		case line, ok := <-s.lines:
			if !ok {
				return "", "", errors.New("Stockfish closed its output during position inspection")
			}
			switch {
			case strings.HasPrefix(line, "Fen:"):
				resultFEN = strings.TrimSpace(strings.TrimPrefix(line, "Fen:"))
				sawFEN = true
			case strings.HasPrefix(line, "Checkers:"):
				checkers = strings.TrimSpace(strings.TrimPrefix(line, "Checkers:"))
				sawCheckers = true
			case strings.TrimSpace(line) == "readyok":
				if !sawFEN || resultFEN == "" {
					return "", "", errors.New("Stockfish did not return a FEN")
				}
				if !sawCheckers {
					return "", "", errors.New("Stockfish did not return checker information")
				}
				return resultFEN, checkers, nil
			}
		case <-timer.C:
			return "", "", fmt.Errorf("timed out after %s inspecting Stockfish position", StockfishCommandWait)
		}
	}
}

func (s *Stockfish) BestMove(fen string, depth int) (string, error) {
	if err := s.Position(fen); err != nil {
		return "", err
	}
	if err := s.Send(fmt.Sprintf("go depth %d", depth)); err != nil {
		return "", err
	}

	for {
		line, err := s.nextLine()
		if err != nil {
			return "", err
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "bestmove" {
			if fields[1] == "(none)" || fields[1] == "0000" {
				return "", nil
			}
			return fields[1], nil
		}
	}
}
