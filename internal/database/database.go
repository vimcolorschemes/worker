package database

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
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
	if strings.HasSuffix(os.Args[0], ".test") {
		// Running in test mode
		return
	}

	connectionString, exists := dotenv.Get("MONGODB_CONNECTION_STRING")
	if !exists {
		log.Panic("Database connection string not found in env")
	}

	clientOptions := options.Client().ApplyURI(connectionString)

	databaseUsername, usernameExists := dotenv.Get("MONGODB_USERNAME")
	databasePassword, passwordExists := dotenv.Get("MONGODB_PASSWORD")
	if usernameExists && databaseUsername != "" && passwordExists && databasePassword != "" {
		credentials := options.Credential{Username: databaseUsername, Password: databasePassword}
		clientOptions.SetAuth(credentials)
	}

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		panic(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		panic(err)
	}

	databaseName, databaseNameExists := dotenv.Get("MONGODB_DATABASE")
	if !databaseNameExists {
		databaseName = "vimcolorschemes"
	}

	database := client.Database(databaseName)
	repositoriesCollection = database.Collection("repositories")
	reportsCollection = database.Collection("reports")
}

// GetRepositories gets all repositories stored in the database
func GetRepositories() []repository.Repository {
	return getRepositories(bson.M{})
}

// GetRepository gets the repository matching the repository key
func GetRepository(repoKey string) (repository.Repository, error) {
	matches := strings.Split(repoKey, "/")

	if len(matches) < 2 {
		return repository.Repository{}, errors.New("key not valid")
	}

	var repo repository.Repository

	ownerName := bson.M{"$regex": matches[0], "$options": "i"}
	name := bson.M{"$regex": matches[1], "$options": "i"}
	err := repositoriesCollection.FindOne(ctx, bson.M{"owner.name": ownerName, "name": name}).Decode(&repo)
	if err != nil {
		return repository.Repository{}, err
	}

	return repo, nil
}

// GetValidRepositories gets repositories stored in the database that are marked as valid
func GetValidRepositories() []repository.Repository {
	return getRepositories(bson.M{"updateValid": true})
}

func getRepositories(filter bson.M) []repository.Repository {
	var repositories []repository.Repository
	cursor, err := repositoriesCollection.Find(ctx, filter)
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

// UpsertRepository updates the repository if it exists, inserts it if not
func UpsertRepository(id int64, updateObject bson.M) {
	filter := bson.M{"_id": id}

	update := bson.M{"$set": updateObject}
	delete(updateObject, "_id")

	upsertOptions := options.Update().SetUpsert(true)

	_, err := repositoriesCollection.UpdateOne(ctx, filter, update, upsertOptions)

	if err != nil {
		log.Printf("Error upserting repository: %s", err)
		panic(err)
	}
}

// CreateReport stores a job report in the database
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
