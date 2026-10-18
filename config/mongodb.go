package config

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// NewMongoDB membuka koneksi MongoDB untuk sistem notifikasi dan pengingat.
func NewMongoDB(cfg *Config) (*mongo.Client, *mongo.Database, error) {
	if cfg.Mongo.URI == "" {
		return nil, nil, fmt.Errorf("mongodb uri is empty")
	}

	clientOpts := options.Client().ApplyURI(cfg.Mongo.URI)
	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize mongodb client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, nil, fmt.Errorf("failed to ping mongodb at %s: %w", cfg.Mongo.URI, err)
	}

	db := client.Database(cfg.Mongo.Database)
	return client, db, nil
}

// CloseMongoDB menutup koneksi MongoDB dengan graceful timeout.
func CloseMongoDB(client *mongo.Client) error {
	if client == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return client.Disconnect(ctx)
}
