package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	pb "github.com/hungpdn/go-grpc-101/pb/sensor/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	conn, _ := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer conn.Close()
	client := pb.NewSensorServiceClient(conn)

	// --- TEST 1: SERVER STREAMING ---
	logger.Info("--- RUNNING SERVER STREAMING ---")
	streamServer, err := client.MonitorTemperature(context.Background(), &pb.MonitorTemperatureRequest{SensorId: "sensor_x1"})
	if err == nil {
		for {
			// Receive data continuously from server
			res, err := streamServer.Recv()
			if err == io.EOF {
				logger.Info("server finished streaming")
				break // Exit loop when server returns nil (end of stream)
			}
			if err != nil {
				logger.Error("stream error", slog.String("error", err.Error()))
				break
			}
			logger.Info("received update", slog.Float64("temp", float64(res.GetTemperature())), slog.String("time", res.GetTimestamp()))
		}
	}

	time.Sleep(2 * time.Second)

	// --- TEST 2: BI-DIRECTIONAL STREAMING ---
	logger.Info("--- RUNNING BI-DIRECTIONAL STREAMING ---")
	streamBiDi, err := client.LiveAlerts(context.Background())
	if err != nil {
		logger.Error("failed to open bidi stream", slog.String("error", err.Error()))
		return
	}

	// Run a background Goroutine to LISTEN for alerts from Server
	waitc := make(chan struct{})
	go func() {
		for {
			alert, err := streamBiDi.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			if err != nil {
				logger.Error("failed to receive alert", slog.String("error", err.Error()))
				return
			}
			logger.Warn("ALERT RECEIVED", slog.String("msg", alert.GetMessage()))
		}
	}()

	// Main Goroutine: CONTINUOUSLY SEND data to Server
	temps := []float32{28.5, 29.5, 30.5, 31.0, 27.0} // Simulate data
	for _, t := range temps {
		logger.Info("sending metric", slog.Float64("value", float64(t)))
		if err := streamBiDi.Send(&pb.LiveAlertsRequest{SensorId: "sensor_x1", Value: t}); err != nil {
			logger.Error("failed to send metric", slog.String("error", err.Error()))
		}
		time.Sleep(500 * time.Millisecond)
	}

	streamBiDi.CloseSend() // Tell Server that Client has finished sending
	<-waitc                // Wait for the receiving Goroutine to finish before exiting the program
}
