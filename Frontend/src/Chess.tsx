import { useEffect, useState } from "react";

export type Move = { from: Square, to: Square };
export type Square = { rank: 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7, file: 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 };
export type Piece = null | "♙" | "♘" | "♗" | "♖" | "♕" | "♔" | "♟" | "♞" | "♝" | "♜" | "♛" | "♚";
export type PromoteTo = "N" | "B" | "R" | "Q";
export type Board = Array<Piece>;
export type ChessColour = null | "white" | "black";
export type GameState = {
    board: Board,
    moves: Array<Move>,
    player: ChessColour,
    gameover: boolean,
    is_player_turn: boolean,
    wins: number,
    losses: number,
    draws: number
};


function readFEN(fen: string): Board {
    const board = new Array<Piece>(64).fill(null)
    let i = 0
    for (let rank = 7; rank >= 0; rank -= 1) {

        let file = 0;
        while (file < 8) {
            if (i == fen.length) {
                console.error(fen, i, "invalid fen: ended unexpectedly");
                return board;
            }
            const char = fen.charAt(i);
            if (char == '1' || char == '2' || char == '3' || char == '4' || char == '5' || char == '6' || char == '7' || char == '8') {
                const count = +char;
                file += count;
                i += 1;
                continue;
            }
            else if (char == 'p') {
                board[rank * 8 + file] = "♟";
            } else if (char == 'P') {
                board[rank * 8 + file] = "♙";
            } else if (char == 'n') {
                board[rank * 8 + file] = "♞";
            } else if (char == 'N') {
                board[rank * 8 + file] = "♘";
            } else if (char == 'b') {
                board[rank * 8 + file] = "♝";
            } else if (char == 'B') {
                board[rank * 8 + file] = "♗";
            } else if (char == 'r') {
                board[rank * 8 + file] = "♜";
            } else if (char == 'R') {
                board[rank * 8 + file] = "♖";
            } else if (char == 'q') {
                board[rank * 8 + file] = "♛";
            } else if (char == 'Q') {
                board[rank * 8 + file] = "♕";
            } else if (char == 'k') {
                board[rank * 8 + file] = "♚";
            } else if (char == 'K') {
                board[rank * 8 + file] = "♔";
            } else {
                console.error(fen, i, rank, file, char.length === 0, "invalid fen: invalid piece" + char);
                return board;
            }
            i += 1;
            file += 1;
        }
        if (file > 8) {
            console.error(fen, i, "invalid fen: overlong rank");
            return board;
        }
        // console.log(file, rank);
        if (rank != 0) {
            const char = fen.charAt(i);
            if (char != '/') {
                console.error(fen, i, file, rank, "invalid fen: expected '/'");
                return board;
            }
            i += 1;
        }
    }
    return board;
}

type ZeroSeven = 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7;

function asZeroToSeven(x: number): ZeroSeven {
    if (x < 0) {
        return 0
    }
    if (x > 7) {
        return 7
    }
    return x as ZeroSeven
}

function readLongAlgebraicNotation(moves: Array<string>): Array<Move> {
    return moves.map((move) => {
        const parsedMove: Move = { from: { rank: 0, file: 0 }, to: { rank: 0, file: 0 } };
        const ACodePoint = "a".charCodeAt(0)
        const OneCodePoint = "1".charCodeAt(0)
        parsedMove.from.file = asZeroToSeven(move.charCodeAt(0) - ACodePoint);
        parsedMove.from.rank = asZeroToSeven(move.charCodeAt(1) - OneCodePoint);
        parsedMove.to.file = asZeroToSeven(move.charCodeAt(2) - ACodePoint);
        parsedMove.to.rank = asZeroToSeven(move.charCodeAt(3) - OneCodePoint);
        return parsedMove;
    })
}

function CanMakeMove(legalMoves: Array<Move>, from: Square, to: Square): boolean {
    for (let index = 0; index < legalMoves.length; index++) {
        const move = legalMoves[index];
        if (move.from.file === from.file
            && move.from.rank === from.rank
            && move.to.file === to.file
            && move.to.rank === to.rank
        ) {
            return true;
        }
    }
    return false
}

export default function ChessInterface() {
    const [ws, setWs] = useState<WebSocket | null>(null)
    const [gameState, setGameState] = useState<GameState>({
        board: Array<Piece>(64).fill(null),
        moves: Array<Move>(),
        player: null,
        gameover: true,
        is_player_turn: false,
        wins: 0,
        losses: 0,
        draws: 0
    })
    const [promoteTo, setPromoteTo] = useState<PromoteTo>("Q")
    const [selectedSquare, setSelectedSquare] = useState<Square | null>(null)

    useEffect(() => {
        const socket = new WebSocket("ws://localhost:8080/socket/");

        socket.onopen = () => {
            console.log("Connected to WebSocket");

        }

        socket.onclose = () => {
            console.log("WebSocket Closed");
        }

        socket.onerror = (err) => {
            console.error("WebSocket Error:", err);
        }

        socket.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);
                if (msg.type === "error") {
                    console.error(msg.message);
                } else if (msg.type === "okay") {
                    console.log(msg.message);
                } else if (msg.type === "update") {
                    console.log(msg.message);
                    console.log(msg);
                    setGameState({
                        board: readFEN(msg.position),
                        moves: readLongAlgebraicNotation(msg.moves),
                        player: msg.player_colour,
                        gameover: msg.gameover,
                        is_player_turn: msg.player_turn,
                        wins: msg.wins,
                        losses: msg.losses,
                        draws: msg.drawsg
                    });
                    setSelectedSquare(null);
                }
            } catch (err) {
                console.log(err)
            }
        };

        setWs(socket);
        return () => socket.close();
    }, []);

    function SendMove(move: string) {
        if (ws && ws.readyState == WebSocket.OPEN) {
            const msg = JSON.stringify({ cmd: "makemove", arg: { "move": move } })
            console.log(msg)
            ws.send(msg)
        }
    }

    function SendResign() {
        if (ws && ws.readyState == WebSocket.OPEN) {
            const msg = JSON.stringify({ cmd: "resign", arg: {} })
            console.log(msg)
            ws.send(msg)
        }
    }

    function SendUndo() {
        if (ws && ws.readyState == WebSocket.OPEN) {
            const msg = JSON.stringify({ cmd: "undo", arg: {} })
            console.log(msg)
            ws.send(msg)
        }
    }

    function SendStartGame(colour: string) {
        if (ws && ws.readyState == WebSocket.OPEN) {
            const msg = JSON.stringify({ cmd: "start", arg: { colour: colour } })
            console.log(msg)
            ws.send(msg)
        }
    }

    const handleSquareClick = (square: Square) => {
        if (!gameState.is_player_turn) return;

        if (selectedSquare !== null && selectedSquare.file === square.file && selectedSquare.rank === square.rank) {
            setSelectedSquare(null);
        } else if (selectedSquare) {
            if (CanMakeMove(gameState.moves, selectedSquare, square)) {
                handleMakeMove(selectedSquare, square);
            }
            setSelectedSquare(null);
        } else {
            setSelectedSquare(square);
            //const moves = getPossibleMoves(board, square);
            //setPossibleMoves(moves);
        }
    }

    const handleMakeMove = (from: Square, to: Square) => {
        const fromfile = 'abcdefgh'[from.file];
        const tofile = 'abcdefgh'[to.file];
        const fromrank = '12345678'[from.rank]
        const torank = '12345678'[to.rank];
        const move = fromfile + fromrank + tofile + torank + promoteTo;
        SendMove(move);
    }

    return <div className="bg-neutral-200 text-neutral-800 flex justify-center items-start p-4">


        <div className="flex flex-col items-start w-[min(80vw,600px)]">

            <div className="flex flex-row relative">

                {gameState.gameover && (
                    <div className="absolute inset-0 flex justify-center items-center z-10">
                        <div className="bg-neutral-200 text-neutral-800 p-4 rounded shadow-md">
                            <OutGameInterface SendStartGame={SendStartGame} />
                        </div>
                    </div>
                )}

                <Board board={gameState.board} selectedSquare={selectedSquare} legalMoves={gameState.moves} handleSquareClick={handleSquareClick} playerColour={gameState.player} />

                {!gameState.gameover && (
                    <div className="ml-6 flex flex-col space-y-4">
                        <InGameInterface SendResign={SendResign} SendUndo={SendUndo} promoteTo={promoteTo} setPromoteTo={setPromoteTo} />
                    </div>
                )}

            </div>



        </div>
    </div>;


}

function PromotionSelector({ promoteTo, setPromoteTo }: { promoteTo: PromoteTo, setPromoteTo: (promoteTo: PromoteTo) => void }) {

    const options: Array<{ label: string, value: PromoteTo }> = [
        { label: "queen", value: "Q" },
        { label: "rook", value: "R" },
        { label: "bishop", value: "B" },
        { label: "knight", value: "N" },
    ]

    return <div className="flex flex-col space-y-2">
        {options.map(({ label, value }) => (
            <button
                onClick={() => setPromoteTo(value)}
                className={`px-4 py-2 rounded transition font-medium
                ${promoteTo === value
                        ? "btn-highlighted"
                        : "btn-default"}
                `}
            >
                {label}
            </button>
        ))}
    </div>;
}

function InGameInterface({ SendResign, SendUndo, promoteTo, setPromoteTo }: { SendResign: () => void, SendUndo: () => void, promoteTo: PromoteTo, setPromoteTo: (promoteTo: PromoteTo) => void }) {
    return <div className="flex flex-col space-y-8">
        <div className="flex flex-col space-y-2">
            <button className="btn-default" onClick={() => SendResign()}>resign</button>
            <button className="btn-default" onClick={() => SendUndo()}>undo</button>
        </div>

        <PromotionSelector promoteTo={promoteTo} setPromoteTo={setPromoteTo} />
    </div>
}



function OutGameInterface({ SendStartGame }: { SendStartGame: (colour: string) => void }) {
    return <div className="flex flex-row justify-center space-x-4">
        <button className="btn-default" onClick={() => SendStartGame("white")}>white</button>
        <button className="btn-default" onClick={() => SendStartGame("random")}>random</button>
        <button className="btn-default" onClick={() => SendStartGame("black")}>black</button>

    </div>
}

function Board({ board, selectedSquare, legalMoves, handleSquareClick, playerColour }: { board: Array<Piece>, selectedSquare: Square | null, legalMoves: Array<Move>, handleSquareClick: (s: Square) => void, playerColour: ChessColour }) {
    const ranks = playerColour === "white" ? [7, 6, 5, 4, 3, 2, 1, 0] : [0, 1, 2, 3, 4, 5, 6, 7];
    const files = [0, 1, 2, 3, 4, 5, 6, 7];
    return <div className="grid grid-cols-8 h-[min(80vw,600px)] aspect-square border border-grey-800 select-none cursor-pointer">
        {ranks.map((rank) => files.map((file) => {

            const squareID = { file: file as ZeroSeven, rank: rank as ZeroSeven };
            const isSelected = selectedSquare !== null && selectedSquare.file === squareID.file && selectedSquare.rank === squareID.rank;
            const isHighlighted = (selectedSquare === null) ? false : CanMakeMove(legalMoves, selectedSquare, squareID);
            return (<SquareElement
                key={file + rank}
                id={squareID}
                piece={board[rank * 8 + file]}
                isSelected={isSelected}
                isHighlighted={isHighlighted}
                onClick={handleSquareClick}
            />);
        })
        )}
    </div>
}


function PieceIcon(piece: Piece) {
    if (piece === null) {
        return <>{''}</>;
    }
    return <>{piece}</>;
}

function SquareElement({ id, piece, isSelected, isHighlighted, onClick }: { id: Square, piece: Piece, isSelected: boolean, isHighlighted: boolean, onClick: (s: Square) => void }) {
    const aCharCode = 'a'.charCodeAt(0);
    const oneCharCode = '1'.charCodeAt(0);
    const isLightSquare = (((id.file - aCharCode) & 1) ^ ((id.rank - oneCharCode) & 1)) === 0;
    // const isWhitePiece =  piece !== null && piece.charAt(0) == "W";
    return (
        <div
            id={'abcdefgh'[id.file] + (id.rank + 1).toString()}
            className={`flex justify-center items-center aspect-square overflow-hidden text-center leading-none
                ${`text-[min(8vw,60px)]`/*isWhitePiece ? "text-green-500" : "text-red-500"*/}
                ${isLightSquare ? "bg-gray-300" : "bg-gray-700"} 
                ${isSelected ? "border-4 border-red-600" : "border border-gray-500"}
                ${isHighlighted ? "bg-red-600" : ""}`}
            onClick={() => onClick(id)}
        >
            {PieceIcon(piece)}
        </div>
    );
}