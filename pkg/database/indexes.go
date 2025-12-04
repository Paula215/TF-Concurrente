package database

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func CreateIndexes() error {
	ctx := context.Background()

	// Índice único para recomendaciones por usuario
	recsIdx := mongo.IndexModel{
		Keys:    bson.M{"userID": 1},
		Options: options.Index().SetUnique(true),
	}

	_, err := RecsCollection().Indexes().CreateOne(ctx, recsIdx)
	if err != nil {
		return fmt.Errorf("error creando índice único en recommendations: %w", err)
	}

	// (Opcional) índice para ratings
	ratingsIdx := mongo.IndexModel{
		Keys:    bson.M{"user_id": 1},
		Options: options.Index(),
	}
	_, err = RatingsCollection().Indexes().CreateOne(ctx, ratingsIdx)
	if err != nil {
		return fmt.Errorf("error creando índice en ratings: %w", err)
	}

	// (Opcional) índice para películas
	moviesIdx := mongo.IndexModel{
		Keys:    bson.M{"movie_id": 1},
		Options: options.Index(),
	}
	_, err = MoviesCollection().Indexes().CreateOne(ctx, moviesIdx)
	if err != nil {
		return fmt.Errorf("error creando índice en movies: %w", err)
	}

	fmt.Println("Índices de MongoDB creados correctamente")
	return nil
}
