package main

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"time"

	pb "github.com/hungpdn/go-grpc-101/pb/sensor/v1"
	"google.golang.org/grpc"
)

type sensorServer struct {
	pb.UnimplementedSensorServiceServer
	logger *slog.Logger
}

// 1. Server Streaming
// Server use loop to Send() data continuously until the end.
func (s *sensorServer) MonitorTemperature(req *pb.MonitorTemperatureRequest, stream pb.SensorService_MonitorTemperatureServer) error {
	s.logger.Info("started temperature monitoring", slog.String("sensor_id", req.GetSensorId()))

	// Simulate sending 5 temperature updates
	for i := 0; i < 5; i++ {
		err := stream.Send(&pb.MonitorTemperatureResponse{
			Temperature: 25.0 + float32(i),
			Timestamp:   time.Now().Format(time.RFC3339),
		})
		if err != nil {
			s.logger.Error("failed to send data", slog.String("error", err.Error()))
			return err // Terminate stream if network error occurs
		}
		time.Sleep(1 * time.Second) // Wait 1 second before sending the next data chunk
	}

	s.logger.Info("monitoring finished")
	return nil // Return nil to signal Server has finished streaming
}

// 3. Bi-directional Streaming
// Run in parallel: Receive stream data (Recv) and Send stream data (Send) at the same time.
func (s *sensorServer) LiveAlerts(stream pb.SensorService_LiveAlertsServer) error {
	s.logger.Info("live alert stream connected")

	for {
		// Recv data from Client
		req, err := stream.Recv()
		if err == io.EOF {
			s.logger.Info("client closed the stream")
			return nil
		}
		if err != nil {
			return err
		}

		s.logger.Info("received metric", slog.Float64("value", float64(req.GetValue())))

		// Business logic: If temperature > 30, immediately send alert to Client
		if req.GetValue() > 30.0 {
			alertMsg := fmt.Sprintf("CRITICAL: Temperature reached %.2f", req.GetValue())
			err = stream.Send(&pb.LiveAlertsResponse{
				SensorId: req.GetSensorId(),
				Message:  alertMsg,
			})
			if err != nil {
				return err
			}
		}
	}
}

// UploadMetrics (Client Streaming) - Ignore to save space in the article,
// but the principle is to use stream.Recv() until io.EOF is encountered, then SendAndClose().
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	listener, _ := net.Listen("tcp", ":50052") // Open port 50052 for lesson 2

	grpcServer := grpc.NewServer()
	pb.RegisterSensorServiceServer(grpcServer, &sensorServer{logger: logger})

	logger.Info("streaming server starting on :50052")
	if err := grpcServer.Serve(listener); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
	}
}
