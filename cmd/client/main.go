package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/gosukretess/battleships/proto/gamepb"
	"github.com/gosukretess/battleships/proto/userpb"
	"github.com/rivo/tview"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var app *tview.Application
var firstTable *tview.TextView
var secondTable *tview.TextView
var header *tview.TextView
var footer *tview.TextView
var inputField *tview.InputField

func main() {
	app = tview.NewApplication()
	grid := initUi()

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("could not connect to gRPC server: %v", err)
	}
	defer conn.Close()

	userClient := userpb.NewUserServiceClient(conn)
	gameClient := gamepb.NewGameServiceClient(conn)

	go gameLoop(&userClient, &gameClient)

	if err := app.SetRoot(grid, true).SetFocus(inputField).Run(); err != nil {
		panic(err)
	}

}

func initUi() *tview.Grid {
	firstTable = tview.NewTextView()
	secondTable = tview.NewTextView()
	header = tview.NewTextView().SetTextAlign(tview.AlignCenter).SetTextColor(tcell.Color226)
	footerGrid := tview.NewGrid().SetRows(0, 1).SetColumns(100).SetBorders(false)
	footer = tview.NewTextView()
	inputField = tview.NewInputField()
	inputField.SetBackgroundColor(tcell.Color19)
	inputField.SetLabel("> ")

	footerGrid.AddItem(footer, 0, 0, 1, 1, 0, 0, false).AddItem(inputField, 1, 0, 1, 1, 0, 0, true)

	grid := tview.NewGrid().
		SetRows(3, 0, 5).
		SetColumns(50, 50).
		SetBorders(true)

	grid.AddItem(firstTable, 1, 0, 1, 1, 0, 100, false).
		AddItem(secondTable, 1, 1, 1, 1, 0, 100, false).
		AddItem(header, 0, 0, 1, 2, 0, 100, false).
		AddItem(footerGrid, 2, 0, 1, 2, 0, 100, false)

	return grid
}

func gameLoop(userClient *userpb.UserServiceClient, gameClient *gamepb.GameServiceClient) {
	var wg sync.WaitGroup

	currentUserId := getUserId(userClient)
	gameId, enemyId, nextUserId := getGameId(gameClient, currentUserId)
	userTurn := false
	if nextUserId == currentUserId {
		userTurn = true
	}

	footer.Clear()

	stream, err := (*gameClient).PlayerMove(context.Background())
	if err != nil {
		log.Fatalf("Cannot connect to start stream: %v", err)
	}

	userShips, _ := (*gameClient).GetShips(context.Background(), &gamepb.GetShipsRequest{
		GameId: gameId,
		UserId: currentUserId,
	})

	moves, _ := (*gameClient).GetMoves(context.Background(), &gamepb.GetMovesRequest{
		GameId: gameId,
		UserId: currentUserId,
	})

	enemyMoves, _ := (*gameClient).GetMoves(context.Background(), &gamepb.GetMovesRequest{
		GameId: gameId,
		UserId: enemyId,
	})

	drawEnemyTable(moves, app, firstTable)
	drawUserTable(userShips, enemyMoves, app, secondTable)

	isUserTurn := make(chan bool, 1)
	isUserTurn <- userTurn

	// RECEIVE EVENT
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			event, err := stream.Recv()
			if err != nil {
				log.Printf("Stream ended or error: %v", err)
				break
			}

			moves, _ := (*gameClient).GetMoves(context.Background(), &gamepb.GetMovesRequest{
				GameId: gameId,
				UserId: currentUserId,
			})

			enemyMoves, _ := (*gameClient).GetMoves(context.Background(), &gamepb.GetMovesRequest{
				GameId: gameId,
				UserId: enemyId,
			})

			drawEnemyTable(moves, app, firstTable)
			drawUserTable(userShips, enemyMoves, app, secondTable)

			if event.UserId2 == currentUserId {
				hit := "TRAFIENIE"
				if event.Type == gamepb.EventType_MISS {
					hit = "PUDŁO"
				}
				writeLog(fmt.Sprintf("Przeciwnik strzelił w (%s,%d) - %s", toLetter(event.X+1), event.Y+1, hit))
				writeLog("Twój ruch! Podaj współrzędne (np. B4)...")
				app.SetFocus(inputField)
				isUserTurn <- true
			} else {
				if event.Type == gamepb.EventType_TAKEN {
					writeLog(fmt.Sprintf("Powtórzony strzał w (%s,%d). Podaj inne współrzędne.", toLetter(event.X+1), event.Y+1))
					isUserTurn <- true
				} else {
					hit := "TRAFIENIE"
					if event.Type == gamepb.EventType_MISS {
						hit = "PUDŁO"
					}
					writeLog(fmt.Sprintf("Strzeliłeś w (%s,%d) - %s", toLetter(event.X+1), event.Y+1, hit))
					writeLog("Czekaj na ruch przeciwnika...")
					app.SetFocus(nil)
					isUserTurn <- false
				}
			}
		}
	}()

	moveChan := make(chan string)
	inputField.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			return
		}
		text := strings.TrimSpace(inputField.GetText())
		inputField.SetText("")
		moveChan <- text
	})

	// SEND MOVE
	wg.Add(1)
	go func() {
		defer wg.Done()

		for turn := range isUserTurn {
			if turn {
				for {
					setHeader("TWOJA TURA")
					app.QueueUpdateDraw(func() {
						header.SetTextColor(tcell.Color40)
					})

					input := <-moveChan
					input = strings.TrimSpace(strings.ToUpper(input))

					if len(input) < 2 {
						writeLog("Nieprawidłowy format. Spróbuj ponownie.")
						continue
					}

					colLetter := input[0]
					rowStr := input[1:]
					x := int32(colLetter - 'A')
					yInt, err := strconv.Atoi(rowStr)
					if err != nil || x < 0 || x > 7 || yInt < 1 || yInt > 8 {
						writeLog("Nieprawidłowe współrzędne. Dozwolone A1–H8.")
						continue
					}
					y := int32(yInt - 1)

					event := &gamepb.GameEvent{
						GameId:  gameId,
						UserId1: currentUserId,
						UserId2: enemyId,
						X:       x,
						Y:       y,
						Type:    gamepb.EventType_MOVE,
					}

					if err := stream.Send(event); err != nil {
						log.Fatalf("Błąd przy wysyłaniu eventu: %v", err)
						continue
					}

					break
				}

			} else {
				setHeader("TURA PRZECIWNIKA")
				app.QueueUpdateDraw(func() {
					header.SetTextColor(tcell.Color196)
				})
			}
		}
	}()

	wg.Wait()
}

func setHeader(text string) {
	var s strings.Builder
	s.WriteString("\n")
	s.WriteString(text)
	setText(header, s.String())
}

func setText(view *tview.TextView, text string) {
	app.QueueUpdateDraw(func() {
		view.SetText(text)
	})
}

func writeLog(text string) {
	app.QueueUpdateDraw(func() {
		currentText := footer.GetText(false)
		footer.SetText(currentText + "\n" + text)
		footer.ScrollToEnd()
	})
}

func toLetter(n int32) string {
	if n < 1 || n > 8 {
		return ""
	}
	return string('A' + rune(n-1))
}

func getUserId(client *userpb.UserServiceClient) string {
	resp, err := (*client).GetUsers(context.Background(), &userpb.GetUsersRequest{})
	if err != nil {
		log.Fatalf("Could not get users: %v", err)
	}

	setHeader("Wybierz użytkownika")
	users := make(map[int]string)
	index := 1
	for _, user := range resp.Users {
		users[index] = user.Id
		app.QueueUpdateDraw(func() {
			fmt.Fprintf(footer, "[%d] %s, Email: %s\n", index, user.Name, user.Email)
		})
		index += 1
	}

	userChan := make(chan string)

	inputField.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			return
		}

		text := strings.TrimSpace(inputField.GetText())
		inputField.SetText("")

		num, err := strconv.Atoi(text)
		if err != nil {
			//setHeader("Niepoprawny numer. Spróbuj ponownie.")
			return
		}

		if val, ok := users[num]; ok {
			inputField.SetDoneFunc(nil)
			userChan <- val
		} else {
			//setHeader("Nie ma użytkownika o takim numerze.")
		}
	})

	return <-userChan
}

func getGameId(client *gamepb.GameServiceClient, currentUserId string) (gameId, enemyId, nextUserId string) {
	resp, err := (*client).GetAllGames(context.Background(), &gamepb.GetAllGamesRequest{})
	if err != nil {
		log.Fatalf("Could not get users: %v", err)
	}

	for _, game := range resp.Games {
		if game.UserId1 == currentUserId {
			return game.Id, game.UserId2, game.NextUser
		} else if game.UserId2 == currentUserId {
			return game.Id, game.UserId1, game.NextUser
		}
	}

	return
}
