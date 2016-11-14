package datastore

import "github.com/jinzhu/gorm"

// Models

// Repository is the representation of a respository in polly
type Repository struct {
	Name           string `json:"name" gorm:"primary_key"`
	GithubID       int    `json:"github_id"`
	OrganizationID string `json:"-"`
}

// InsertRepository inserts the user into the database
func InsertRepository(db *gorm.DB, repo *Repository) error {
	return db.Debug().Create(repo).Error
}

// FindRepositoryByName returns the user with the specified name
func FindRepositoryByName(db *gorm.DB, name string) (*Repository, error) {
	var repo Repository
	err := db.Where("name = ?", name).First(&repo).Error
	return &repo, err
}
