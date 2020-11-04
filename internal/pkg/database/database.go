package database

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository struct {
	ID   primitive.ObjectID `bson:"_id,omitempty"`
	Name string             `bson:"name,omitempty"`
}

var connectionString string
var repositoriesCollection *mongo.Collection
var ctx = context.TODO()

func init() {
	log.Print("Loading .env file")
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func init() {
	connectionString, exists := os.LookupEnv("MONGODB_CONNECTION_STRING")
	if !exists {
		log.Panic("No connection string provided")
	}

	clientOptions := options.Client().ApplyURI(connectionString)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		panic(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		panic(err)
	}

	repositoriesCollection = client.Database("colorschemes").Collection("repositories")
}

func GetRepositories() []Repository {
	var repositories []Repository
	cursor, err := repositoriesCollection.Find(ctx, bson.M{})
	if err != nil {
		log.Print("here")
		panic(err)
	}
	if err = cursor.All(ctx, &repositories); err != nil {
		log.Print("or here")
		panic(err)
	}
	return repositories
}
