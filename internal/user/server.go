package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/gosukretess/battleships/proto/userpb"
)

type Server struct {
	userpb.UnimplementedUserServiceServer
	store *Store
}

func NewServer(store *Store) *Server {
	return &Server{store: store}
}

func (s *Server) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.GetUserResponse, error) {

	name, email, err := s.store.GetUser(req.GetId())

	if err != nil {
		return nil, err
	}

	return &userpb.GetUserResponse{
		User: &userpb.User{
			Id:    req.GetId(),
			Name:  name,
			Email: email,
		},
	}, nil
}

func (s *Server) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	id := uuid.New().String()
	err := s.store.CreateUser(id, req.GetName(), req.GetEmail())
	if err != nil {
		return nil, err
	}

	return &userpb.CreateUserResponse{
		User: &userpb.User{
			Id:    id,
			Name:  req.GetName(),
			Email: req.GetEmail(),
		},
	}, nil
}

func (s *Server) GetUsers(ctx context.Context, req *userpb.GetUsersRequest) (*userpb.GetUsersResponse, error) {
	users, err := s.store.GetUsers()

	if err != nil {
		return nil, err
	}

	var pbUsers []*userpb.User
	for _, u := range users {
		pbUsers = append(pbUsers, &userpb.User{
			Id:    u.Id,
			Name:  u.Name,
			Email: u.Email,
		})
	}

	return &userpb.GetUsersResponse{
		Users: pbUsers,
	}, nil
}
