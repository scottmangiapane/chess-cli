package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

const DefaultEnginePath = "/opt/homebrew/bin/stockfish"

type Engine struct {
	cmd   *exec.Cmd
	input io.WriteCloser
	lines chan string
}

func StartEngine(path string, threads, hashMB int) (*Engine, error) {
	cmd := exec.Command(path)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start Stockfish: %w", err)
	}

	e := &Engine{
		cmd:   cmd,
		input: stdin,
		lines: make(chan string, 1024),
	}

	go func() {
		defer close(e.lines)
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 4096), 1024*1024)

		for scanner.Scan() {
			e.lines <- scanner.Text()
		}
	}()

	fail := func(err error) (*Engine, error) {
		e.Close()
		return nil, err
	}

	// The UCI handshake consumes the engine's startup output, so it
	// never gets printed in the user's terminal.
	if err := e.Send("uci"); err != nil {
		return fail(err)
	}
	if err := e.waitFor("uciok"); err != nil {
		return fail(err)
	}

	if err := e.Send(fmt.Sprintf("setoption name Threads value %d", threads)); err != nil {
		return fail(err)
	}
	if err := e.Send(fmt.Sprintf("setoption name Hash value %d", hashMB)); err != nil {
		return fail(err)
	}

	if err := e.Ready(); err != nil {
		return fail(err)
	}
	if err := e.Send("ucinewgame"); err != nil {
		return fail(err)
	}
	if err := e.Ready(); err != nil {
		return fail(err)
	}

	return e, nil
}

func (e *Engine) Send(command string) error {
	_, err := fmt.Fprintln(e.input, command)
	return err
}

func (e *Engine) nextLine() (string, error) {
	line, ok := <-e.lines
	if !ok {
		return "", errors.New("Stockfish closed its output")
	}
	return line, nil
}

func (e *Engine) waitFor(expected string) error {
	for {
		line, err := e.nextLine()
		if err != nil {
			return err
		}
		if strings.TrimSpace(line) == expected {
			return nil
		}
	}
}

func (e *Engine) Ready() error {
	if err := e.Send("isready"); err != nil {
		return err
	}
	return e.waitFor("readyok")
}

func (e *Engine) Close() {
	if e == nil || e.cmd == nil || e.cmd.Process == nil {
		return
	}
	_ = e.Send("quit")
	_ = e.input.Close()
	_ = e.cmd.Wait()
}

// Position sets the position to a FEN, optionally followed by UCI moves.
func (e *Engine) Position(fen string, moves ...string) error {
	command := "position fen " + fen
	if len(moves) > 0 {
		command += " moves " + strings.Join(moves, " ")
	}
	return e.Send(command)
}

// LegalMoves uses Stockfish's perft command to enumerate legal moves.
func (e *Engine) LegalMoves(fen string) ([]string, error) {
	if err := e.Position(fen); err != nil {
		return nil, err
	}
	if err := e.Send("go perft 1"); err != nil {
		return nil, err
	}

	var moves []string

	for {
		line, err := e.nextLine()
		if err != nil {
			return nil, err
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
	}
}

// Inspect returns the FEN and checker squares for a position reached
// by applying an optional UCI move to the supplied FEN.
func (e *Engine) Inspect(fen string, moves ...string) (string, string, error) {
	if err := e.Position(fen, moves...); err != nil {
		return "", "", err
	}
	if err := e.Send("d"); err != nil {
		return "", "", err
	}
	if err := e.Send("isready"); err != nil {
		return "", "", err
	}

	var resultFEN, checkers string

	for {
		line, err := e.nextLine()
		if err != nil {
			return "", "", err
		}

		switch {
		case strings.HasPrefix(line, "Fen:"):
			resultFEN = strings.TrimSpace(strings.TrimPrefix(line, "Fen:"))
		case strings.HasPrefix(line, "Checkers:"):
			checkers = strings.TrimSpace(strings.TrimPrefix(line, "Checkers:"))
		case line == "readyok":
			if resultFEN == "" {
				return "", "", errors.New("Stockfish did not return a FEN")
			}
			return resultFEN, checkers, nil
		}
	}
}

// BestMove searches the current position to the requested depth.
func (e *Engine) BestMove(fen string, depth int) (string, error) {
	if err := e.Position(fen); err != nil {
		return "", err
	}
	if err := e.Send(fmt.Sprintf("go depth %d", depth)); err != nil {
		return "", err
	}

	for {
		line, err := e.nextLine()
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
