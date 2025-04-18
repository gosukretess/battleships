package main

import (
	"log"
	"net"

	"github.com/gosukretess/battleships/internal"
	"github.com/gosukretess/battleships/proto/gamepb"
	"github.com/gosukretess/battleships/proto/userpb"
	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()

	srv, err := internal.InitializeServers("database.db")
	if err != nil {
		log.Fatalf("failed to init server: %v", err)
	}

	userpb.RegisterUserServiceServer(grpcServer, srv.UserServer)
	gamepb.RegisterGameServiceServer(grpcServer, srv.GameServer)

	log.Println("gRPC runs at port :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
