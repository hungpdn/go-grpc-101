package main

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	pb "github.com/hungpdn/go-grpc-101/pb/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	conn, _ := grpc.NewClient("localhost:50063", grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer conn.Close()
	client := pb.NewUserServiceClient(conn)

	// Fire 5 concurrent requests instantly
	var wg sync.WaitGroup
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go func(reqNum int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			_, err := client.GetUser(ctx, &pb.GetUserRequest{UserId: "usr_1"})
			if err != nil {
				logger.Error("request failed", slog.Int("req", reqNum), slog.String("error", err.Error()))
			} else {
				logger.Info("request succeeded", slog.Int("req", reqNum))
			}
		}(i)
	}
	wg.Wait()
}
