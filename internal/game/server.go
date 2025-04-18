package game

import (
	"context"
	"io"
	"log"
	"math/rand"
	"sync"
	"time"

	"slices"

	"github.com/gosukretess/battleships/proto/gamepb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	gamepb.UnimplementedGameServiceServer
	store   *Store
	mu      sync.Mutex
	streams []gamepb.GameService_PlayerMoveServer
}

func NewServer(store *Store) *Server {
	return &Server{
		store:   store,
		streams: make([]gamepb.GameService_PlayerMoveServer, 0),
	}
}

// Just 12 ships with size 1 - this is just a game draft
func (s *Server) CreateGame(_ context.Context, req *gamepb.CreateGameRequest) (*gamepb.CreateGameResponse, error) {
	type Coords struct {
		x int
		y int
	}

	contains := func(coords []Coords, c Coords) bool {
		for _, existing := range coords {
			if existing.x == c.x && existing.y == c.y {
				return true
			}
		}
		return false
	}

	gameDto, err := s.store.CreateGame(req.GetUserId1(), req.GetUserId2())
	if err != nil {
		return nil, err
	}

	usersShips := make(map[string][]Coords)
	usersShips[req.GetUserId1()] = make([]Coords, 0, 12)
	usersShips[req.GetUserId2()] = make([]Coords, 0, 12)
	for _, userId := range []string{req.GetUserId1(), req.GetUserId2()} {
		// Generate unique 12 coords
		for len(usersShips[userId]) < 12 {
			x := rand.Intn(8)
			y := rand.Intn(8)
			ship := Coords{
				x: x,
				y: y,
			}

			if !contains(usersShips[userId], ship) {
				usersShips[userId] = append(usersShips[userId], ship)
			}
		}

		for _, coords := range usersShips[userId] {
			s.store.AddShip(gameDto.Id, userId, coords.x, coords.y)
		}
	}

	parsedTime, err := time.Parse(time.RFC3339Nano, gameDto.Created)
	createdTimestamp := timestamppb.New(parsedTime)

	return &gamepb.CreateGameResponse{
		Game: &gamepb.Game{
			Id:       gameDto.Id,
			UserId1:  gameDto.UserId1,
			UserId2:  gameDto.UserId2,
			Created:  createdTimestamp,
			NextUser: gameDto.NextUser,
		},
	}, err
}

func (s *Server) GetAllGames(_ context.Context, req *gamepb.GetAllGamesRequest) (*gamepb.GetAllGamesResponse, error) {
	games, err := s.store.GetGames()

	if err != nil {
		return nil, err
	}

	var pbGames []*gamepb.Game
	for _, g := range games {
		parsedTime, _ := time.Parse(time.RFC3339Nano, g.Created)
		createdTimestamp := timestamppb.New(parsedTime)
		pbGames = append(pbGames, &gamepb.Game{
			Id:       g.Id,
			UserId1:  g.UserId1,
			UserId2:  g.UserId2,
			Created:  createdTimestamp,
			NextUser: g.NextUser,
		})
	}

	return &gamepb.GetAllGamesResponse{
		Games: pbGames,
	}, nil
}

func (s *Server) PlayerMove(stream gamepb.GameService_PlayerMoveServer) error {
	s.mu.Lock()
	if !containsStream(s.streams, stream) {
		s.streams = append(s.streams, stream)
	}
	s.mu.Unlock()

	var gameId string

	for {
		event, err := stream.Recv()
		if err == io.EOF {
			log.Println("Stream ended")
			break
		}
		if err != nil {
			log.Printf("Error: %v", err)
			break
		}

		gameId = event.GameId
		log.Printf("[Game] Received event: %+v", event)

		if event.Type == gamepb.EventType_MOVE {
			eventType := gamepb.EventType_MISS

			taken, _ := s.store.AreCoordsTaken(gameId, event.UserId1, int(event.X), int(event.Y))

			log.Printf("%v", taken)
			if taken {
				eventType = gamepb.EventType_TAKEN
			} else {
				hit, err := s.store.Move(gameId, event.UserId1, int(event.X), int(event.Y))
				if err != nil {
					continue
				}

				if hit {
					eventType = gamepb.EventType_HIT
				}
			}

			responseEvent := gamepb.GameEvent{
				GameId:  gameId,
				UserId1: event.UserId1,
				UserId2: event.UserId2,
				X:       event.X,
				Y:       event.Y,
				Type:    eventType,
			}

			log.Printf("[Game] Sending event: %+v", responseEvent)

			for _, userStream := range s.streams {
				userStream.Send(&responseEvent)
			}
		}
	}

	s.mu.Lock()
	s.streams = removeStream(s.streams, stream)
	s.mu.Unlock()

	return nil
}

func (s *Server) GetShips(ctx context.Context, req *gamepb.GetShipsRequest) (*gamepb.GetShipsResponse, error) {
	ships, err := s.store.GetShips(req.GetGameId(), req.GetUserId())
	if err != nil {
		return nil, err
	}

	var result []*gamepb.Ship
	for _, ship := range ships {
		result = append(result, &gamepb.Ship{
			GameId: ship.GameId,
			UserId: ship.UserId,
			X:      int32(ship.X),
			Y:      int32(ship.Y),
		})
	}

	return &gamepb.GetShipsResponse{Ships: result}, nil
}

func (s *Server) GetMoves(ctx context.Context, req *gamepb.GetMovesRequest) (*gamepb.GetMovesResponse, error) {
	moves, err := s.store.GetMoves(req.GetGameId(), req.GetUserId())
	if err != nil {
		return nil, err
	}

	var result []*gamepb.Move
	for _, move := range moves {
		result = append(result, &gamepb.Move{
			GameId: move.GameId,
			UserId: move.UserId,
			X:      int32(move.X),
			Y:      int32(move.Y),
			Hit:    move.Hit,
		})
	}

	return &gamepb.GetMovesResponse{
		Moves: result,
	}, nil
}

func containsStream(streams []gamepb.GameService_PlayerMoveServer, s gamepb.GameService_PlayerMoveServer) bool {
	return slices.Contains(streams, s)
}

func removeStream(streams []gamepb.GameService_PlayerMoveServer, s gamepb.GameService_PlayerMoveServer) []gamepb.GameService_PlayerMoveServer {
	newStreams := []gamepb.GameService_PlayerMoveServer{}
	for _, str := range streams {
		if str != s {
			newStreams = append(newStreams, str)
		}
	}
	return newStreams
}
