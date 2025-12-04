package database

// -----------------------------------------------------------
// DOCUMENTO: Recomendación generada para un usuario
// Colección: recommendations
// -----------------------------------------------------------

type RecommendedItem struct {
	MovieID   string  `bson:"movie_id" json:"movie_id"`
	Predicted float64 `bson:"predicted" json:"predicted"`
}

type RecommendationDocument struct {
	UserID        string            `bson:"userID" json:"userID"`
	Recommended   []RecommendedItem `bson:"recommended" json:"recommended"`
	LatencyMS     int64             `bson:"latency_ms" json:"latency_ms"`
	TimestampUnix int64             `bson:"timestamp_unix" json:"timestamp_unix"`
}

// -----------------------------------------------------------
// DOCUMENTO: Log del proceso distribuido
// Colección: logs
// -----------------------------------------------------------

type LogDocument struct {
	UserID        string `bson:"user_id" json:"user_id"`
	NodeCount     int    `bson:"node_count" json:"node_count"`
	LatencyMS     int64  `bson:"latency_ms" json:"latency_ms"`
	TimestampUnix int64  `bson:"timestamp" json:"timestamp"`
}

// -----------------------------------------------------------
// DOCUMENTO: Movie
// Colección: movies
// -----------------------------------------------------------

type Movie struct {
	MovieID string   `bson:"movie_id" json:"movie_id"`
	Title   string   `bson:"title" json:"title"`
	Genres  []string `bson:"genres" json:"genres"`
}

// -----------------------------------------------------------
// DOCUMENTO: Rating
// Colección: ratings
// -----------------------------------------------------------

type Rating struct {
	UserID  string  `bson:"user_id" json:"user_id"`
	MovieID string  `bson:"movie_id" json:"movie_id"`
	Rating  float64 `bson:"rating" json:"rating"`
}
