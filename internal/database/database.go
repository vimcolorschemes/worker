package database

import (
	"context"
	"log"

	"github.com/vimcolorschemes/worker/internal/dotenv"

	gogithub "github.com/google/go-github/v32/github"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository struct {
	ID    int64 `bson:"_id,omitempty"`
	Name  string
	Owner struct {
		Name      string
		AvatarURL string
	}
}

var ctx = context.TODO()
var repositoriesCollection *mongo.Collection

func init() {
	connectionString := dotenv.Get("MONGODB_CONNECTION_STRING", true)
	clientOptions := options.Client().ApplyURI(connectionString)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		panic(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		panic(err)
	}

	repositoriesCollection = client.Database("vimcolorschemes").Collection("repositories")
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

func UpsertRepositories(repositories []*gogithub.Repository) {
	for _, repository := range repositories {
		log.Print("Upserting ", *repository.Name)

		filter := bson.M{"_id": *repository.ID}

		updateObject := getRepositoryUpdateObject(repository)
		update := bson.M{"$set": updateObject}

		upsertOptions := options.Update().SetUpsert(true)

		_, err := repositoriesCollection.UpdateOne(ctx, filter, update, upsertOptions)

		if err != nil {
			panic(err)
		}
	}
}

func getRepositoryUpdateObject(repository *gogithub.Repository) bson.M {
		return bson.M{
			"_id":              *repository.ID,
			"owner.name":       *repository.Owner.Login,
			"owner.avatarURL": *repository.Owner.AvatarURL,
			"name":             *repository.Name,
		}
}
