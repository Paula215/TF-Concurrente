Sistema de Recomendación Distribuido y Concurrente
Go + MongoDB + Redis

Este proyecto implementa un sistema de recomendación de películas basado en Filtrado Colaborativo (User-Based KNN). Utiliza una arquitectura distribuida escrita en Go y aprovecha concurrencia y caching para procesar grandes volúmenes de datos de forma eficiente.

Características principales

Arquitectura distribuida con API orquestadora y nodos de cálculo vía TCP.

Concurrencia mediante goroutines y channels.

Algoritmo User-Based KNN basado en similitud de coseno.

Caching de resultados mediante Redis.

Persistencia y consultas históricas usando MongoDB.

Despliegue completo con contenedores mediante Docker Compose.

Estructura del proyecto
├── cmd/
│   ├── api/        # API HTTP (orquestador)
│   ├── nodo/       # Worker TCP (cálculo distribuido KNN)
│   ├── loadmongo/  # Importación de CSVs a MongoDB
│   └── bench/      # Herramientas de benchmarking
├── internal/
│   ├── analisis/   # Reportes estadísticos del dataset
│   ├── cleaning/   # Limpieza concurrente de datos raw
│   └── knn/        # Coseno, vecinos y predicción
├── pkg/
│   ├── database/   # Drivers y modelos MongoDB
│   └── network/    # Protocolos TCP API <-> Workers
├── docker/         # Dockerfiles y docker-compose
├── data/           # Dataset raw/clean (ignorado por git)
└── go.mod

Requisitos previos

Docker y Docker Compose

Go 1.22+ (solo para ejecución local de herramientas)

Dataset MovieLens en formato .dat en data/raw/

Archivos esperados:

ratings.dat
movies.dat
tags.dat

Instalación y ejecución
1. Limpieza y preparación de datos
go run internal/cleaning/clean.go


Salida generada en data/clean/.

2. Despliegue del sistema completo
cd docker
docker-compose up --build


Se desplegarán:

MongoDB

Redis

API en puerto 8080

Nodos TCP

Servicio de carga automática hacia MongoDB

Endpoints principales

Base URL: http://localhost:8080

Método	Endpoint	Descripción
GET	/recommend/{userID}	Genera recomendaciones
GET	/ratings/{userID}	Muestra historial del usuario
GET	/movies	Lista todas las películas
GET	/movies/genre/{genre}	Filtra por género

Ejemplo:

curl http://localhost:8080/recommend/1

Flujo de procesamiento

Se consulta recomendación desde el cliente.

La API valida caché en Redis.

Si no existe en caché, consulta MongoDB.

Si no hay datos almacenados:

La API corta el dataset en particiones.

Envía a nodos TCP.

Cada nodo calcula similitud de coseno.

La API consolida los resultados.

Se almacena respuesta en Redis y MongoDB.

Se retorna la respuesta final al cliente.

Herramientas adicionales
Análisis exploratorio
go run internal/analisis/analisis.go


Genera resultados en: /analysis/

Benchmarking de concurrencia
go run cmd/bench/main.go -workers="1,2,4,8" -sample=100


Permite evaluar diferencias entre número de workers y tiempo de procesamiento.

Tecnologías involucradas

Go

MongoDB con driver oficial

Redis usando go-redis/v9

TCP sockets con serialización gob

Docker Compose

Autor

Proyecto desarrollado para el curso de Programación Concurrente y Distribuida.

Ayrton Samaniego
Paula Mancilla
