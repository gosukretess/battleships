//go:build wireinject
// +build wireinject

package internal

import (
	"github.com/google/wire"
	"github.com/gosukretess/battleships/internal/game"
	"github.com/gosukretess/battleships/internal/user"
)

type Server struct {
	UserServer *user.Server
	GameServer *game.Server
}

func NewServer(userServer *user.Server, gameServer *game.Server) *Server {
	return &Server{
		UserServer: userServer,
		GameServer: gameServer,
	}
}

func InitializeServers(dbPath string) (*Server, error) {
	wire.Build(
		user.NewStore,
		user.NewServer,
		game.NewStore,
		game.NewServer,
		NewServer,
	)
	return nil, nil
}
