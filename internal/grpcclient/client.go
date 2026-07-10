package grpcclient

import (
	"context"
	"log/slog"
	"math"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"

	"github.com/argus/ecs-fargate-agent/internal/config"
	pb "github.com/argus/ecs-fargate-agent/internal/hermespb"
)

type Client struct {
	cfg    config.Config
	logger *slog.Logger
}

func New(cfg config.Config, logger *slog.Logger) *Client {
	return &Client{cfg: cfg, logger: logger}
}

func (c *Client) Send(ctx context.Context, batch *pb.Batch) error {
	var last error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * 250 * time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}
		if err := c.sendOnce(ctx, batch); err != nil {
			last = err
			c.logger.Warn("grpc send attempt failed", "attempt", attempt+1, "error", err)
			continue
		}
		return nil
	}
	return last
}

func (c *Client) sendOnce(ctx context.Context, batch *pb.Batch) error {
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		dialCtx,
		c.cfg.HermesAddress(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)),
		grpc.WithBlock(),
	)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pb.NewIngestClient(conn)
	stream, err := client.StreamBatches(ctx)
	if err != nil {
		return err
	}
	if err := stream.Send(batch); err != nil {
		return err
	}
	_, err = stream.CloseAndRecv()
	return err
}
