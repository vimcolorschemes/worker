package database

import (
	"context"
	"log"

	"github.com/vimcolorschemes/worker/internal/dotenv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository struct {
	ID   primitive.ObjectID `bson:"_id,omitempty"`
	Name string             `bson:"name,omitempty"`
}

var repositoriesCollection *mongo.Collection
var ctx = context.TODO()

func init() {
	clientOptions := options.Client().ApplyURI(dotenv.Get("MONGODB_CONNECTION_STRING"))

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
