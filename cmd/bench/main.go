package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"pcd-pc4/internal/knn"
)

func main() {
	ratingsPath := flag.String("ratings", "data/clean/ratings.csv", "ruta al CSV de ratings")
	sampleUsers := flag.Int("sample", 100, "numero de usuarios de prueba (primeros N usuarios)")
	workersList := flag.String("workers", "1,2,4,8", "lista de workers separados por coma")
	outCSV := flag.String("out", "bench_results.csv", "archivo de salida CSV")
	k := flag.Int("k", 50, "k vecinos")
	flag.Parse()

	// Cargar datos
	fmt.Println("Cargando ratings desde:", *ratingsPath)
	userRatings := knn.LoadUserRatings(*ratingsPath)
	if len(userRatings) == 0 {
		fmt.Println("No se cargaron ratings")
		return
	}

	// Crear lista de usuarios de prueba
	users := make([]string, 0, len(userRatings))
	for u := range userRatings {
		users = append(users, u)
		if len(users) >= *sampleUsers {
			break
		}
	}
	if len(users) == 0 {
		fmt.Println("No hay usuarios para sample")
		return
	}

	// Parse workers
	var workers []int
	for _, s := range splitAndTrim(*workersList) {
		v, _ := strconv.Atoi(s)
		if v > 0 {
			workers = append(workers, v)
		}
	}

	// Preparar CSV
	f, err := os.Create(*outCSV)
	if err != nil {
		fmt.Println("No se pudo crear CSV:", err)
		return
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	w.Write([]string{"workers", "sample_users", "elapsed_seconds", "timestamp_unix"})

	// Ejecutar tests
	for _, wcnt := range workers {
		start := time.Now()
		// Llamada principal: reuse tu función para correr KNN paralelo
		knn.RunKNNBenchmark(userRatings, users, wcnt, *k) // función helper que describo abajo
		elapsed := time.Since(start).Seconds()
		fmt.Printf("workers=%d elapsed=%.3f s\n", wcnt, elapsed)
		w.Write([]string{strconv.Itoa(wcnt), strconv.Itoa(len(users)), fmt.Sprintf("%.6f", elapsed), strconv.FormatInt(time.Now().Unix(), 10)})
		w.Flush()
	}
	fmt.Println("Benchmark finalizado. Resultados en", *outCSV)
}

func splitAndTrim(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := s[start:i]
			// trim spaces
			j1 := 0
			j2 := len(part)
			for j1 < j2 && part[j1] == ' ' {
				j1++
			}
			for j2 > j1 && part[j2-1] == ' ' {
				j2--
			}
			if j2 > j1 {
				out = append(out, part[j1:j2])
			}
			start = i + 1
		}
	}
	return out
}
