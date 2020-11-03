package import_runner

import (
	"context"
	"log"

	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository struct {
	ID   primitive.ObjectID `bson:"_id,omitempty"`
	Name string             `bson:"name,omitempty"`
}

func Run() {
	connectionString, exists := os.LookupEnv("MONGODB_CONNECTION_STRING")

	if !exists {
		log.Print("No connection string provided")
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionString))
	if err != nil {
		panic(err)
	}
	defer client.Disconnect(ctx)

	database := client.Database("colorschemes")

	repositoriesCollection := database.Collection("repositories")

	var repositories []Repository

	cursor, err := repositoriesCollection.Find(ctx, bson.M{})
	if err != nil {
		panic(err)
	}

	if err = cursor.All(ctx, &repositories); err != nil {
		panic(err)
	}

	log.Println(repositories)
}
