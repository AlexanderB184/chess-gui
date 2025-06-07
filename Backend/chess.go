package main

import (
	Chess "chessgui/piecemeal"

	"github.com/gorilla/websocket"
)

type Game struct {
	Player       Chess.Colour      `json:"player_colour"`
	Position     *Chess.ChessState `json:"position"`
	IsPlayerTurn bool              `json:"player_turn"`
	GameOver     bool              `json:"gameover"`
	LegalMoves   []Chess.Move      `json:"moves"`
	LastPlayed   Chess.Move        `json:"last_move"`
	Undos        int
}

type Session struct {
	Game

	PlayerWins   uint `json:"wins"`
	ComputerWins uint `json:"losses"`
	Draws        uint `json:"draws"`

	conn *websocket.Conn
}

func NewGame(Player Chess.Colour) Game {
	var LegalMoves []Chess.Move
	var StartPosition = Chess.NewGame()
	var IsPlayerTurn = Player == Chess.WHITE
	if IsPlayerTurn {
		LegalMoves = StartPosition.LegalMoves()
	} else {
		LegalMoves = []Chess.Move{}
	}
	return Game{
		Player:       Player,
		Position:     StartPosition,
		IsPlayerTurn: IsPlayerTurn,
		GameOver:     false,
		LegalMoves:   LegalMoves,
		LastPlayed:   Chess.Move{},
		Undos:        0,
	}

}

func NewSession(conn *websocket.Conn) *Session {
	return &Session{
		conn:         conn,
		PlayerWins:   0,
		ComputerWins: 0,
		Draws:        0,
		Game:         Game{GameOver: true},
	}
}

func (game *Session) MakeMove(move Chess.Move) error {
	if err := game.Position.MakeMove(move); err != nil {
		return err
	}

	game.LastPlayed = move

	gameover := game.Position.IsGameover()

	if gameover != Chess.ONGOING {
		game.GameOver = true
		winner := game.Position.GetWinner()
		if winner == game.Player {
			game.PlayerWins++
		} else if winner != 0 {
			game.ComputerWins++
		} else {
			game.Draws++
		}
	}

	game.IsPlayerTurn = game.Position.WhoToMove() == game.Player

	if game.IsPlayerTurn && !game.GameOver {
		game.LegalMoves = game.Position.LegalMoves()
	} else {
		game.LegalMoves = []Chess.Move{}
	}

	return nil
}
