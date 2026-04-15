package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/icleary5/SafelyYouChallenge/handler"
	"github.com/icleary5/SafelyYouChallenge/model"
)

func main() {
	store, err := model.NewMemoryStoreFromCSV("devices.csv")
	if err != nil {
		log.Fatal("error loading devices: ", err)
	}

	// Note: host and port are fixed per the API specification.
	if err := setupRouter(store).Run("127.0.0.1:6733"); err != nil {
		log.Fatal("error starting server: ", err)
	}
}

// setupRouter creates a Gin engine with all routes registered against the
// provided store. Keeping this in package main lets tests call it without
// introducing a circular import.
func setupRouter(store model.Store) *gin.Engine {
	r := gin.Default()
	handler.New(store).RegisterRoutes(r)
	return r
}
