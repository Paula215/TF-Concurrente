package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"pcd-pc4/pkg/database"
)

func main() {
	uri := "mongodb://pcd-pc4_mongo:27017"
	err := database.Connect(uri)
	if err != nil {
		log.Fatal("Error conectando a MongoDB:", err)
	}

	fmt.Println("Conectado a MongoDB")

	loadMovies("data/clean/movies.csv")
	loadRatings("data/clean/ratings.csv")

	fmt.Println("Carga completada.")
}

// ------------------------
// Cargar movies.csv
// ------------------------

func loadMovies(path string) {
	f, _ := os.Open(path)
	defer f.Close()

	r := csv.NewReader(f)
	rows, _ := r.ReadAll()

	col := database.MoviesCollection()

	for i, row := range rows {
		if i == 0 || len(row) < 3 {
			continue
		}

		doc := database.Movie{
			MovieID: row[0],
			Title:   row[1],
			Genres:  strings.Split(row[2], "|"),
		}

		_, err := col.InsertOne(context.Background(), doc)
		if err != nil {
			fmt.Println("Error insertando movie:", err)
		}
	}

	fmt.Println("Movies cargadas en MongoDB (géneros como lista)")
}

// ------------------------
// Cargar ratings.csv
// ------------------------

func loadRatings(path string) {
	f, _ := os.Open(path)
	defer f.Close()

	r := csv.NewReader(f)
	rows, _ := r.ReadAll()

	col := database.RatingsCollection()

	for i, row := range rows {
		if i == 0 || len(row) < 3 {
			continue
		}

		rval, _ := strconv.ParseFloat(row[2], 64)

		doc := database.Rating{
			UserID:  row[0],
			MovieID: row[1],
			Rating:  rval,
		}

		_, err := col.InsertOne(context.Background(), doc)
		if err != nil {
			fmt.Println("Error insertando rating:", err)
		}
	}

	fmt.Println("Ratings cargados en MongoDB")
}
