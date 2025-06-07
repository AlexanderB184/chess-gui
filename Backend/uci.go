package main

import (
	"bufio"
	Chess "chessgui/piecemeal"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type ClockInfo struct {
	wtime uint64
	btime uint64
	winc  uint64
	binc  uint64
}

type StopCond struct {
	nodeLimitNds  uint64
	timeLimitMs   uint64
	depthLimitPly uint64
	mateInPly     uint64
}

type BotInterface struct {
	path   string
	cond   *sync.Cond
	active bool
	quiet  bool
	cmd    *exec.Cmd
	to     *bufio.Writer
	from   *bufio.Reader
	name   string
	author string
}

func (bot *BotInterface) setActive() {
	bot.cond.L.Lock()
	for bot.active {
		bot.cond.Wait()
	}
	bot.active = true
	bot.cond.L.Unlock()

}

func (bot *BotInterface) setInactive() {
	bot.cond.L.Lock()
	bot.active = false
	bot.cond.Broadcast()
	bot.cond.L.Unlock()
}

func (bot *BotInterface) Send(args ...string) error {
	cmd := strings.Join(args, " ") + "\n"
	if !bot.quiet {
		log.Printf("[GUI] %q\n", cmd)
	}
	if _, err := bot.to.WriteString(cmd); err != nil {
		log.Print(err)
		return err
	}
	if err := bot.to.Flush(); err != nil {
		log.Print(err)
		return err
	}
	return nil

}

func (bot *BotInterface) Receive() (args []string, err error) {
	response, err := bot.from.ReadString('\n')
	if err != nil {
		log.Print(err)
		return
	}
	if !bot.quiet {
		log.Printf("[%v] %q\n", bot.name, response)
	}
	return strings.Fields(response), nil
}

func RestartBot(bot *BotInterface) error {
	if bot.cmd != nil {
		if err := bot.cmd.Process.Kill(); err != nil {
			return err
		}

		if err := bot.cmd.Wait(); err != nil {
			return err
		}
	}

	cmd := exec.Command(bot.path)

	*bot = BotInterface{

		active: false,
		quiet:  false,
		cmd:    cmd,
		name:   "BOT",
	}

	if stdinPipe, err := cmd.StdinPipe(); err != nil {
		return err
	} else {
		bot.to = bufio.NewWriter(stdinPipe)
	}

	if stdoutPipe, err := cmd.StdoutPipe(); err != nil {
		return err
	} else {
		bot.from = bufio.NewReader(stdoutPipe)
	}
	cmd.Start()
	bot.Send("uci")
	timeLimit := 500 * time.Millisecond

	select {

	default:
		for {
			msg, err := bot.Receive()

			if err != nil {
				log.Print(err)
				return err
			}
			if len(msg) >= 1 && msg[0] == "uciok" {
				return nil
			}
			if len(msg) >= 2 && msg[0] == "id" {
				if msg[1] == "name" {
					bot.name = strings.Join(msg[2:], " ")
				} else if msg[1] == "author" {
					bot.author = strings.Join(msg[2:], " ")
				}
			}
			if len(msg) >= 2 && msg[0] == "option" {

			}
		}
	case <-time.After(timeLimit):
		log.Printf("bot failed to initialize in time")
		return fmt.Errorf("bot failed to initialize in time")
	}
}

func NewBot(botPath string) (*BotInterface, error) {
	cmd := exec.Command(botPath)

	bot := &BotInterface{
		cond:   sync.NewCond(&sync.Mutex{}),
		path:   botPath,
		active: false,
		quiet:  false,
		cmd:    cmd,
		name:   "BOT",
	}

	if stdinPipe, err := cmd.StdinPipe(); err != nil {
		return nil, err
	} else {
		bot.to = bufio.NewWriter(stdinPipe)
	}

	if stdoutPipe, err := cmd.StdoutPipe(); err != nil {
		return nil, err
	} else {
		bot.from = bufio.NewReader(stdoutPipe)
	}
	cmd.Start()
	bot.Send("uci")
	timeLimit := 500 * time.Millisecond

	select {

	default:
		for {
			msg, err := bot.Receive()

			if err != nil {
				log.Print(err)
				return nil, err
			}
			if len(msg) >= 1 && msg[0] == "uciok" {
				return bot, nil
			}
			if len(msg) >= 2 && msg[0] == "id" {
				if msg[1] == "name" {
					bot.name = strings.Join(msg[2:], " ")
				} else if msg[1] == "author" {
					bot.author = strings.Join(msg[2:], " ")
				}
			}
			if len(msg) >= 2 && msg[0] == "option" {

			}
		}
	case <-time.After(timeLimit):
		log.Printf("bot failed to initialize in time")
		return nil, fmt.Errorf("bot failed to initialize in time")
	}
}

// this sends only the minimum number of moves in the move stack for checking draws,
// keeping the size of the message as small as possible while retaining all necessary information.
// this is important since the chess bot only has a buffer size of 1024 bytes.
func (bot *BotInterface) LoadPosition(chessState *Chess.ChessState) error {
	// unmake to last irreversible move or root, save fen, then store each move after to move text
	cmd := []string{"position", "fen"}
	moveStack := []Chess.Move{}
	lastIrreversibleMove := chessState.Ply() - chessState.HalfMoveClock()
	for chessState.Ply() > lastIrreversibleMove && chessState.Ply() > 0 {
		moveStack = append(moveStack, chessState.PrevMove())
		chessState.UnmakeMove()
	}

	cmd = append(cmd, chessState.Fen())

	if len(moveStack) > 0 {
		cmd = append(cmd, "moves")
		for i := len(moveStack) - 1; i >= 0; i-- {
			//log.Print(moveStack[i])
			cmd = append(cmd, moveStack[i].LongAlgebraicNotation())
			chessState.MakeMove(moveStack[i])
		}
	}

	return bot.Send(cmd...)
}

// handles all the possible features the UCI interface supports
func (bot *BotInterface) Go(ponder bool, searchmoves *[]Chess.Move, clockinfo *ClockInfo, stopcond *StopCond) error {
	cmd := []string{"go"}
	if ponder {
		cmd = append(cmd, "ponder")
	}
	if searchmoves != nil {
		cmd = append(cmd, "searchmoves")
		for _, move := range *searchmoves {
			cmd = append(cmd, fmt.Sprint(move))
		}
	}
	if clockinfo != nil {
		cmd = append(cmd, "wtime", fmt.Sprint(clockinfo.wtime))
		cmd = append(cmd, "btime", fmt.Sprint(clockinfo.btime))
		cmd = append(cmd, "winc", fmt.Sprint(clockinfo.winc))
		cmd = append(cmd, "binc", fmt.Sprint(clockinfo.binc))
	}
	if stopcond != nil {
		if stopcond.depthLimitPly > 0 {
			cmd = append(cmd, "depth", fmt.Sprint(stopcond.depthLimitPly))
		}
		if stopcond.nodeLimitNds > 0 {
			cmd = append(cmd, "nodes", fmt.Sprint(stopcond.nodeLimitNds))
		}
		if stopcond.timeLimitMs > 0 {
			cmd = append(cmd, "movetime", fmt.Sprint(stopcond.timeLimitMs))
		}
		if stopcond.mateInPly > 0 {
			cmd = append(cmd, "matein", fmt.Sprint(stopcond.mateInPly))
		}
		if stopcond.depthLimitPly == 0 && stopcond.nodeLimitNds == 0 &&
			stopcond.timeLimitMs == 0 && stopcond.mateInPly == 0 {
			cmd = append(cmd, "infinite")
		}
	} else {
		// default to 100ms time limit
		cmd = append(cmd, "movetime", "100")
	}

	return bot.Send(cmd...)
}

func (bot *BotInterface) BestMove(chessState *Chess.ChessState) (Chess.Move, error) {
	bot.setActive()
	defer bot.setInactive()

	timeLimitMs := 100
	timeLimit := 110 * time.Millisecond

	bot.LoadPosition(chessState)
	bot.Go(false, nil, nil, &StopCond{timeLimitMs: uint64(timeLimitMs)})

	select {
	case <-time.After(timeLimit):
		bot.Send("stop")
		// maybe extra stuff error/crash handling after this
		return Chess.Move{}, fmt.Errorf("bot failed to terminate on time")
	default:
		for {
			msg, err := bot.Receive()
			if err != nil {
				return Chess.Move{}, err
			}
			if len(msg) >= 2 && msg[0] == "bestmove" {
				move, _, err := chessState.ParseMove([]byte(msg[1]))
				return move, err
			}
			if len(msg) >= 1 && msg[0] == "info" {

			}
		}
	}
}

func (bot *BotInterface) Stop() error {
	return bot.Send("stop")
}

func (bot *BotInterface) Kill() error {
	if err := bot.Send("quit"); err != nil {
		return err
	}
	select {
	case <-time.After(500 * time.Millisecond):
		if err := bot.cmd.Process.Kill(); err != nil {
			return err
		}
	default:
		if err := bot.cmd.Wait(); err != nil {
			return err
		}
	}
	return nil
}
