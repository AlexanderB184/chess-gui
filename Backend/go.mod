module chessgui/backend

require github.com/gorilla/websocket v1.5.3

require chessgui/piecemeal v1.0.0

replace chessgui/piecemeal => ../PieceMeal/bindings/go


go 1.23.0
