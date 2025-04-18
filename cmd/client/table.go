package main

import (
	"fmt"
	"strings"

	"github.com/gosukretess/battleships/proto/gamepb"
	"github.com/rivo/tview"
)

func drawUserTable(userShips *gamepb.GetShipsResponse, enemyMoves *gamepb.GetMovesResponse, app *tview.Application, view *tview.TextView) {
	var b strings.Builder
	columns := []rune{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H'}

	b.WriteString("Twoja plansza:\n")
	b.WriteString("   ")
	for _, col := range columns {
		fmt.Fprintf(&b, " %c ", col)
	}
	b.WriteString("\n")

	for newY := 0; newY < 8; newY++ {
		fmt.Fprintf(&b, "%2d ", newY+1)
		for newX := 0; newX < 8; newX++ {
			if move, ok := tryGetMove(enemyMoves.Moves, int32(newX), int32(newY)); ok {
				if move.Hit {
					b.WriteString("[X]")
				} else {
					b.WriteString("[~]")
				}
			} else if hasShipAt(userShips.Ships, int32(newX), int32(newY)) {
				b.WriteString("[\u25A0]")
			} else {
				b.WriteString("[ ]")
			}

		}
		b.WriteString("\n")
	}

	app.QueueUpdateDraw(func() {
		view.SetText(b.String())
	})
}

func drawEnemyTable(moves *gamepb.GetMovesResponse, app *tview.Application, view *tview.TextView) {
	var b strings.Builder
	columns := []rune{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H'}

	b.WriteString("Plansza przeciwnika:\n")
	b.WriteString("   ")
	for _, col := range columns {
		fmt.Fprintf(&b, " %c ", col)
	}
	b.WriteString("\n")

	for newY := 0; newY < 8; newY++ {
		fmt.Fprintf(&b, "%2d ", newY+1)
		for newX := 0; newX < 8; newX++ {
			move, ok := tryGetMove(moves.Moves, int32(newX), int32(newY))
			if ok {
				if move.Hit {
					b.WriteString("[X]")
				} else {
					b.WriteString("[o]")
				}
			} else {
				b.WriteString("[ ]")
			}
		}
		b.WriteString("\n")
	}

	app.QueueUpdateDraw(func() {
		view.SetText(b.String())
	})
}

func hasShipAt(ships []*gamepb.Ship, x, y int32) bool {
	for _, ship := range ships {
		if ship.X == x && ship.Y == y {
			return true
		}
	}
	return false
}

func tryGetMove(moves []*gamepb.Move, x, y int32) (*gamepb.Move, bool) {
	for _, move := range moves {
		if move.X == x && move.Y == y {
			return move, true
		}
	}
	return nil, false
}
