// Package database owns the MongoDB connection lifecycle.
package database

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"ginmongo/internal/config"
)

type Mongo struct {
	client *mongo.Client
	dbName string
}

// Connect dials MongoDB and verifies the connection with a ping.
func Connect(ctx context.Context, cfg config.MongoConfig) (*Mongo, error) {
	ctx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	m := &Mongo{client: client, dbName: cfg.Database}
	if err := m.Ping(ctx); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo ping: %w", err)
	}
	return m, nil
}

// DB returns the application database handle repositories are built from.
func (m *Mongo) DB() *mongo.Database {
	return m.client.Database(m.dbName)
}

// Ping satisfies the health handler's Pinger interface.
func (m *Mongo) Ping(ctx context.Context) error {
	return m.client.Ping(ctx, readpref.Primary())
}

func (m *Mongo) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}
