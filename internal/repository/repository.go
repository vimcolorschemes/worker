package repository

import (
	gogithub "github.com/google/go-github/v32/github"
	"go.mongodb.org/mongo-driver/bson"
)

func UniquifyRepositories(repositories []*gogithub.Repository) []*gogithub.Repository {
	keys := make(map[int64]bool)
	unique := []*gogithub.Repository{}

	for _, repository := range repositories {
		if _, value := keys[*repository.ID]; !value {
			keys[*repository.ID] = true
			unique = append(unique, repository)
		}
	}

	return unique
}

func GetRepositoryUpdateObject(repository *gogithub.Repository) bson.M {
	return bson.M{
		"_id":             *repository.ID,
		"owner.name":      *repository.Owner.Login,
		"owner.avatarURL": *repository.Owner.AvatarURL,
		"name":            *repository.Name,
	}
}
