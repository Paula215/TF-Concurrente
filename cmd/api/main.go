package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"pcd-pc4/internal/knn"
	"pcd-pc4/pkg/database"
	"pcd-pc4/pkg/network"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	userRatings map[string]map[string]float64
	movieTitles map[string]string

	// Nombres de contenedor Docker
	nodes = []string{
		"pcd-pc4_nodo1:9000",
		"pcd-pc4_nodo2:9001",
	}

	// Redis
	redisClient *redis.Client
	ctx         = context.Background()
)

const (
	K        = 50
	TopN     = 10
	RedisTTL = 10 * time.Minute
)

func main() {
	fmt.Println("Cargando datos limpios de MovieLens...")

	userRatings = knn.LoadUserRatings("data/clean/ratings.csv")
	movieTitles = knn.LoadMovieTitles("data/clean/movies.csv")

	if len(userRatings) == 0 {
		log.Fatal("No se pudieron cargar ratings.")
	}

	// --------------------------------------------------
	// Conexión a MongoDB
	// --------------------------------------------------

	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://pcd-pc4_mongo:27017"
	}

	fmt.Println("Conectando a MongoDB en:", uri)

	if err := database.Connect(uri); err != nil {
		log.Fatal("Error conectando a MongoDB: ", err)
	}

	fmt.Println("Conexión a MongoDB lista.")
	// Crear índices
	if err := database.CreateIndexes(); err != nil {
		log.Fatal("Error creando índices en MongoDB:", err)
	}

	// --------------------------------------------------
	// Conexión a Redis (cache)
	// --------------------------------------------------

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "pcd-pc4_redis:6379"
	}
	redisClient = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		// No fatal: permitimos continuar sin cache, pero avisamos
		log.Fatalf("Error conectando a Redis en %s: %v", redisAddr, err)
	}
	fmt.Println("Conexión a Redis lista:", redisAddr)

	// --------------------------------------------------
	// Iniciar servidor HTTP
	// --------------------------------------------------

	fmt.Println("API distribuida escuchando en puerto 8080...")

	http.HandleFunc("/recommend/", handleRecommendUser)
	http.HandleFunc("/ratings/", handleUserRatings)
	http.HandleFunc("/movies/genre/", handleMoviesByGenre)
	http.HandleFunc("/movies", handleAllMovies)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

// -----------------------------------------------------------
// VERIFICAR SI EXISTE RECOMENDACIÓN EN MONGODB
// -----------------------------------------------------------

func checkExistingRecommendationInMongo(user string) (bool, []knn.Recommended, error) {
	col := database.RecsCollection()

	filter := map[string]interface{}{
		"userID": user,
	}

	var existingDoc database.RecommendationDocument
	err := col.FindOne(context.Background(), filter).Decode(&existingDoc)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			// No existe documento para este usuario
			return false, nil, nil
		}
		// Error en la consulta
		return false, nil, err
	}

	// Convertir de database.RecommendedItem a knn.Recommended
	var recs []knn.Recommended
	for _, item := range existingDoc.Recommended {
		recs = append(recs, knn.Recommended{
			MovieID:   item.MovieID,
			Predicted: item.Predicted,
		})
	}

	return true, recs, nil
}

// -----------------------------------------------------------
// ENDPOINT: GET /recommend/:userID
// -----------------------------------------------------------

func handleRecommendUser(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Path[len("/recommend/"):]
	if user == "" {
		http.Error(w, "Debe especificar un usuario", 400)
		return
	}

	if _, ok := userRatings[user]; !ok {
		http.Error(w, "Usuario no encontrado", 404)
		return
	}

	cacheKey := "recs:" + user

	// 1) VERIFICAR PRIMERO SI YA EXISTE EN MONGODB
	// (Solo si Redis está disponible, usamos cache primero)
	if redisClient != nil {
		if cached, err := redisClient.Get(ctx, cacheKey).Result(); err == nil {
			// Cache hit: devolvemos directamente el JSON almacenado
			fmt.Println("CACHE HIT para", user)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(cached))
			return
		} else if err != redis.Nil {
			// error distinto de key not found: loguear y continuar (no fatal)
			fmt.Println("Aviso: error al consultar Redis:", err)
		} else {
			// err == redis.Nil -> key no existe (cache miss)
			fmt.Println("CACHE MISS para", user)
		}
	}

	// 2) VERIFICAR SI YA EXISTE EN MONGODB ANTES DE CALCULAR
	exists, existingRecs, err := checkExistingRecommendationInMongo(user)
	if err != nil {
		fmt.Println("Error verificando MongoDB:", err)
		// Continuamos con el cálculo en caso de error
	} else if exists {
		fmt.Println("Recomendación ya existe en MongoDB para usuario:", user)

		// Si existe en MongoDB pero no en Redis, la guardamos en Redis
		if redisClient != nil {
			jsonBytes, err := json.Marshal(existingRecs)
			if err == nil {
				redisClient.Set(ctx, cacheKey, string(jsonBytes), RedisTTL)
				fmt.Println("Copiado de MongoDB a Redis para usuario:", user)
			}
		}

		// Respondemos con las recomendaciones existentes
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(existingRecs)
		return
	}

	// 3) Si no existe en MongoDB, calcular recomendaciones
	start := time.Now()
	recs, err := distributedRecommendation(user)
	if err != nil {
		http.Error(w, "Error en recomendación: "+err.Error(), 500)
		return
	}
	latency := time.Since(start).Milliseconds()

	// 4) Guardar en MongoDB (asíncrono)
	go saveRecommendationToMongo(user, recs, latency)

	// Serializar recomendaciones a JSON
	jsonBytes, err := json.Marshal(recs)
	if err != nil {
		fmt.Println("Error al serializar recomendaciones:", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(recs)
		return
	}

	// 5) Guardar en Redis (si está disponible) con TTL
	if redisClient != nil {
		if err := redisClient.Set(ctx, cacheKey, string(jsonBytes), RedisTTL).Err(); err != nil {
			fmt.Println("Advertencia: no se pudo escribir en Redis:", err)
		} else {
			fmt.Println("Guardado en cache Redis:", cacheKey)
		}
	}

	// Responder
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonBytes)
}

// -----------------------------------------------------------
// ENDPOINT: GET /ratings/:userID
// -----------------------------------------------------------

func handleUserRatings(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Path[len("/ratings/"):]
	if user == "" {
		http.Error(w, "Debe proporcionar userID", 400)
		return
	}

	col := database.RatingsCollection()

	filter := bson.M{"user_id": user}

	cursor, err := col.Find(context.Background(), filter)
	if err != nil {
		http.Error(w, "Error consultando MongoDB", 500)
		return
	}
	defer cursor.Close(context.Background())

	var results []database.Rating
	if err := cursor.All(context.Background(), &results); err != nil {
		http.Error(w, "Error leyendo datos", 500)
		return
	}

	// Respuesta JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// -----------------------------------------------------------
// ENDPOINT: GET /movies/genre/:name
// -----------------------------------------------------------

func handleMoviesByGenre(w http.ResponseWriter, r *http.Request) {
	genre := r.URL.Path[len("/movies/genre/"):]
	if genre == "" {
		http.Error(w, "Debe especificar un género", 400)
		return
	}

	col := database.MoviesCollection()

	// Filtro: buscar películas cuyo array de géneros contenga el valor indicado
	filter := bson.M{
		"genres": genre,
	}

	cursor, err := col.Find(context.Background(), filter)
	if err != nil {
		http.Error(w, "Error consultando MongoDB", 500)
		return
	}
	defer cursor.Close(context.Background())

	var movies []database.Movie
	if err := cursor.All(context.Background(), &movies); err != nil {
		http.Error(w, "Error leyendo datos", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movies)
}

// -----------------------------------------------------------
// ENDPOINT: GET /movies
// -----------------------------------------------------------
func handleAllMovies(w http.ResponseWriter, r *http.Request) {
	col := database.MoviesCollection()

	cursor, err := col.Find(context.Background(), bson.M{})
	if err != nil {
		http.Error(w, "Error consultando MongoDB", 500)
		return
	}
	defer cursor.Close(context.Background())

	var movies []database.Movie
	if err := cursor.All(context.Background(), &movies); err != nil {
		http.Error(w, "Error leyendo datos", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movies)
}

// -----------------------------------------------------------
// PROCESO DISTRIBUIDO: API → nodos ML
// -----------------------------------------------------------

func distributedRecommendation(targetUser string) ([]knn.Recommended, error) {
	// Dividir usuarios en chunks (uno por nodo)
	chunks := splitUsersIntoChunks(userRatings, len(nodes))

	allNeighbors := []network.NeighborResult{}

	for i, chunk := range chunks {
		addr := nodes[i]

		partial, err := sendTaskToNode(addr, targetUser, chunk)
		if err != nil {
			return nil, err
		}

		allNeighbors = append(allNeighbors, partial...)
	}

	// Selección global de top K vecinos
	topK := knn.TopK(allNeighbors, K)

	// Predecir ratings
	recs := knn.PredictRatings(targetUser, userRatings, topK)

	return knn.TopNRecommendations(recs, TopN), nil
}

// -----------------------------------------------------------
// TCP: enviar tarea a cada nodo
// -----------------------------------------------------------

func sendTaskToNode(addr, target string, chunk map[string]map[string]float64) ([]network.NeighborResult, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println("Error conectando a nodo", addr, ":", err)
		return nil, err
	}
	defer conn.Close()

	req := network.TaskRequest{
		TargetUser: target,
		UserChunk:  chunk,
		K:          K,
	}

	if err := network.Send(conn, req); err != nil {
		return nil, err
	}

	var resp network.TaskResponse
	if err := network.Receive(conn, &resp); err != nil {
		return nil, err
	}

	return resp.PartialNeighbors, nil
}

// -----------------------------------------------------------
// GUARDAR RECOMENDACIÓN EN MONGODB
// -----------------------------------------------------------

func saveRecommendationToMongo(user string, recs []knn.Recommended, latencyMS int64) {
	col := database.RecsCollection()

	// 1. Verificar si ya existe un documento para este userID
	filter := bson.M{"userID": user}

	var existing database.RecommendationDocument
	err := col.FindOne(context.Background(), filter).Decode(&existing)

	if err == nil {
		// Documento existente → NO guardar otra vez
		fmt.Println("Mongo: recomendación YA EXISTE, no se guardará duplicado para usuario:", user)
		return
	}

	// Si el error NO es ErrNoDocuments → error real
	if err != nil && err != mongo.ErrNoDocuments {
		fmt.Println("Mongo: error inesperado al verificar existencia:", err)
		return
	}

	// 2. Convertir recomendaciones
	items := make([]database.RecommendedItem, 0, len(recs))
	for _, r := range recs {
		items = append(items, database.RecommendedItem{
			MovieID:   r.MovieID,
			Predicted: r.Predicted,
		})
	}

	// 3. Crear documento nuevo
	doc := database.RecommendationDocument{
		UserID:        user,
		Recommended:   items,
		LatencyMS:     latencyMS,
		TimestampUnix: time.Now().Unix(),
	}

	// 4. Insertar
	_, err = col.InsertOne(context.Background(), doc)
	if err != nil {
		fmt.Println("Mongo: error al insertar recomendación:", err)
		return
	}

	fmt.Println("Mongo: recomendación insertada correctamente para usuario:", user)
}

// -----------------------------------------------------------
// Dividir usuarios en N partes
// -----------------------------------------------------------

func splitUsersIntoChunks(data map[string]map[string]float64, parts int) []map[string]map[string]float64 {
	chunks := make([]map[string]map[string]float64, parts)

	for i := 0; i < parts; i++ {
		chunks[i] = make(map[string]map[string]float64)
	}

	i := 0
	for user, ratings := range data {
		idx := i % parts
		chunks[idx][user] = ratings
		i++
	}

	return chunks
}
