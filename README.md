# Sistema de Recomendación Distribuido y Concurrente  
### Go + MongoDB + Redis

Este proyecto implementa un sistema de recomendación de películas basado en Filtrado Colaborativo (User-Based KNN). Utiliza una arquitectura distribuida escrita en Go y aprovecha concurrencia y caching para procesar grandes volúmenes de datos de forma eficiente.

---

## Características principales

- Arquitectura distribuida con API orquestadora y nodos de cálculo vía TCP.  
- Concurrencia mediante goroutines y channels.  
- Algoritmo User-Based KNN basado en similitud de coseno.  
- Caching de resultados mediante Redis.  
- Persistencia y consultas históricas usando MongoDB.  
- Despliegue completo con contenedores mediante Docker Compose.

---

## Requisitos previos

- Docker y Docker Compose  
- Go 1.22+ (solo para ejecución local de herramientas)  
- Dataset MovieLens en formato `.dat` en la carpeta `data/raw/`

Archivos esperados:

ratings.dat
movies.dat
tags.dat

yaml
Copy code

---

## Instalación y ejecución

### 1. Limpieza y preparación de datos

Ejecutar la herramienta de limpieza:

```bash
go run internal/cleaning/clean.go
Esto generará archivos limpios en data/clean/.
```

2. Despliegue del sistema completo
Ejecutar desde el directorio /docker:

```bash
cd docker
docker-compose up --build
```

Servicios levantados:

MongoDB

Redis

API Gateway en puerto 8080

Nodos de procesamiento TCP

Servicio de carga automática hacia MongoDB

Endpoints principales
Base URL: http://localhost:8080

Método	Endpoint	Descripción
```bash
GET	/recommend/{userID}	Genera recomendaciones
GET	/ratings/{userID}	Retorna ratings del usuario
GET	/movies	Lista todas las películas
GET	/movies/genre/{genre}	Filtra películas por género
```
Ejemplo de uso:

```bash
curl http://localhost:8080/recommend/1
Flujo de procesamiento distribuido
El cliente solicita una recomendación.
```
La API revisa si existe en Redis.

Si no existe, revisa MongoDB.

Si tampoco existe allí:

Divide el dataset en bloques.

Envía tareas a nodos TCP.

Cada nodo calcula similitud de coseno concurrentemente.

La API agrega resultados parciales y predice ratings.

El resultado es almacenado en Redis y MongoDB.

Se devuelve al cliente.

Herramientas complementarias
Análisis exploratorio del dataset
bash
Copy code
go run internal/analisis/analisis.go
Genera CSVs en /analysis con métricas y estadísticas.

Benchmarking de concurrencia
```bash
go run cmd/bench/main.go -workers="1,2,4,8" -sample=100
```
Permite comparar tiempos de procesamiento.

Tecnologías usadas
Lenguaje Go

MongoDB con driver oficial

Redis (go-redis/v9)

TCP Sockets con gob

Docker Compose

Autor
Ayton Samaniego
Paula Mancilla
