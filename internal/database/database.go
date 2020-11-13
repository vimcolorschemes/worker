package database

import (
	"context"
	"log"
	"time"

	"github.com/vimcolorschemes/worker/internal/dotenv"
	"github.com/vimcolorschemes/worker/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ctx = context.TODO()
var repositoriesCollection *mongo.Collection
var reportsCollection *mongo.Collection

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

	database := client.Database("vimcolorschemes")
	repositoriesCollection = database.Collection("repositories")
	reportsCollection = database.Collection("reports")
}

func GetRepositories() []repository.Repository {
	var repositories []repository.Repository
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

func UpsertRepository(id int64, updateObject bson.M) {
	filter := bson.M{"_id": id}

	update := bson.M{"$set": updateObject}

	upsertOptions := options.Update().SetUpsert(true)

	_, err := repositoriesCollection.UpdateOne(ctx, filter, update, upsertOptions)

	if err != nil {
		log.Printf("Error upserting repository: %s", err)
		panic(err)
	}
}

func CreateReport(job string, elapsedTime float64, data bson.M) {
	object := bson.M{
		"date":        time.Now(),
		"job":         job,
		"elapsedTime": elapsedTime,
		"data":        data,
	}
	_, err := reportsCollection.InsertOne(ctx, object, &options.InsertOneOptions{})

	if err != nil {
		log.Printf("Error creating report: %s", err)
		panic(err)
	}
}
