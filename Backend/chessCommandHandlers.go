package main

import (
	Chess "chessgui/piecemeal"
	json "encoding/json"
	"io"
	"log"
	"math/rand"
	http "net/http"

	websocket "github.com/gorilla/websocket"
)

type BaseResponse struct {
	Type string `json:"type"`
	Msg  string `json:"message"`
}

type BoardUpdate struct {
	Type         string   `json:"type"`
	Msg          string   `json:"message"`
	Player       string   `json:"player_colour"`
	Position     string   `json:"position"`
	IsPlayerTurn bool     `json:"player_turn"`
	GameOver     bool     `json:"gameover"`
	LegalMoves   []string `json:"moves"`
	LastMove     string   `json:"last_move"`
	PlayerWins   uint     `json:"wins"`
	ComputerWins uint     `json:"losses"`
	Draws        uint     `json:"draws"`
}

type ClientMessage struct {
	Cmd string          `json:"cmd"`
	Arg json.RawMessage `json:"arg"`
}

type MakeMoveCmd struct {
	Move string `json:"move"`
}

type StartCmd struct {
	Colour string `json:"colour"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func runChess(w http.ResponseWriter, r *http.Request) {

	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Print(err)
		return
	}

	defer conn.Close()

	log.Print("connected!")

	session := NewSession(conn)

	for {
		msgType, reader, err := conn.NextReader()

		if err != nil {
			log.Print(err)
			break
		}

		switch msgType {
		default:
			continue
		case websocket.PingMessage:
			conn.WriteMessage(websocket.PongMessage, nil)
		case websocket.PongMessage:
			log.Print("Received Pong from client")
		case websocket.TextMessage:
			rawMsg, err := io.ReadAll(reader)
			if err != nil {
				log.Print(err)
				continue
			}
			handleMessage(rawMsg, session)
		}
	}
}

var chessCmdHandlers = map[string]func(*json.RawMessage, *Session){
	"makemove": handleMakeMove,
	"resign":   handleResign,
	"start":    handleStartGame,
	"undo":     handleUndo,
}

func handleMessage(rawMsg []byte, session *Session) {
	var msg ClientMessage

	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		SendError(session, "Invalid JSON.")
		return
	}

	if handler, exists := chessCmdHandlers[msg.Cmd]; exists {
		handler(&msg.Arg, session)
	} else {
		SendError(session, "Invalid Command.")
		log.Print(msg.Cmd)
	}

}

func handleStartGame(arg *json.RawMessage, game *Session) {
	if !game.GameOver {
		SendError(game, "Cannot Reset While Game Active.")
		return
	}

	game.Game = NewGame(game.Player)
	var startArg StartCmd

	if err := json.Unmarshal(*arg, &startArg); err != nil {
		return
	}

	if startArg.Colour == "black" {
		game.Player = Chess.BLACK
	} else if startArg.Colour == "white" {
		game.Player = Chess.WHITE
	} else if startArg.Colour == "random" {
		if rand.Int()%2 == 0 {
			game.Player = Chess.BLACK
		} else {
			game.Player = Chess.WHITE
		}
	} else {
		SendError(game, "Invalid Colour.")
	}

	game.Game = NewGame(game.Player)

	SendGameState(game, "Started Game.")

	if !game.IsPlayerTurn {
		go handleBotMove(game)
	}
}

func handleResign(arg *json.RawMessage, game *Session) {
	if game.GameOver {
		SendError(game, "Cannot Resign While Game Inactive.")
		return
	}
	game.ComputerWins++
	game.GameOver = true
	game.LegalMoves = []Chess.Move{}
	game.IsPlayerTurn = false

	SendGameState(game, "Resigned.")
}

func handleMakeMove(arg *json.RawMessage, game *Session) {
	if game.GameOver {
		SendError(game, "Game Is Over.")
		return
	}

	if game.Position.WhoToMove() != game.Player {
		SendError(game, "Not your turn.")
		return
	}

	var moveCmd MakeMoveCmd

	if err := json.Unmarshal(*arg, &moveCmd); err != nil {
		log.Print(err)
		return
	}

	move, _, err := game.Position.ParseMove([]byte(moveCmd.Move))

	if err != nil {
		SendError(game, "Invalid Move.")
		return
	}

	if err := game.MakeMove(move); err != nil {
		SendError(game, "Illegal Move.")
		return
	}

	SendGameState(game, "Played Move.")

	// No need to get bot response when bot has been checkmated. The game is over.
	if !game.Position.IsCheckmate() {
		go handleBotMove(game)
	}

}

func handleUndo(arg *json.RawMessage, game *Session) {
	
	if game.GameOver {
		SendError(game, "Game Is Over.")
		return
	}
	if game.Position.WhoToMove() != game.Player {
		SendError(game, "Not your turn.")
		return
	}
	if game.Position.Ply() < 2 {
		SendError(game, "Cannot Undo.")
		return
	}
	if err := game.Position.UnmakeMove(); err != nil {
		// game state is now corrupted extra handling is needed
		game.GameOver = true
		log.Print(err)
		SendError(game, "Cannot Undo.")
		return
	}
	if err := game.Position.UnmakeMove(); err != nil {
		// game state is now corrupted extra handling is needed
		game.GameOver = true
		log.Print(err)
		SendError(game, "Cannot Undo.")
		return
	}
	game.LegalMoves = game.Position.LegalMoves()
	game.LastPlayed = game.Position.PrevMove()
	game.Undos++
	SendGameState(game, "Undo.")
}

func SendError(game *Session, msg string) {

	err := game.conn.WriteJSON(BaseResponse{
		Type: "error",
		Msg:  msg,
	})

	if err != nil {
		log.Print(err)
		return
	}
}

func SendOkay(game *Session, msg string) {

	err := game.conn.WriteJSON(BaseResponse{
		Type: "okay",
		Msg:  msg,
	})

	if err != nil {
		log.Print(err)
		return
	}
}

func moveList(moveList []Chess.Move) []string {
	out := []string{}
	for _, move := range moveList {
		out = append(out, move.LongAlgebraicNotation())
	}
	return out
}

func SendGameState(game *Session, msg string) {
	var player string

	if game.GameOver || game.Player == Chess.WHITE {
		player = "white"
	} else {
		player = "black"
	}

	err := game.conn.WriteJSON(BoardUpdate{
		Type:         "update",
		Msg:          msg,
		Player:       player,
		Position:     game.Position.Fen(),
		IsPlayerTurn: game.IsPlayerTurn,
		LastMove:     game.LastPlayed.LongAlgebraicNotation(),
		GameOver:     game.GameOver,
		LegalMoves:   moveList(game.LegalMoves),
		PlayerWins:   game.PlayerWins,
		ComputerWins: game.ComputerWins,
		Draws:        game.Draws,
	})

	if err != nil {
		log.Print(err)
		return
	}
}

func handleBotMove(game *Session) {
	botmove, err := Bot.BestMove(game.Position)
	if err != nil {
		log.Printf("Bot Error: %v", err)
		return // bot failed something wrong
	}
	game.MakeMove(botmove)
	SendGameState(game, "Bot Move.")
}
