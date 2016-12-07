package datastore

import (
	"time"

	"github.com/jinzhu/gorm"
)

// Models

// Server represents a backend gerrit instance
type Server struct {
	ID             string `gorm:"AUTO_INCREMENT"`
	IPAddr         string
	OrganizationID int
	CreatedAt      time.Time
}

// GetServerForOrganization returns the server for this org
func GetServerForOrganization(db *gorm.DB, orgName string) (*Server, error) {
	var server Server
	err := db.Find(&server, "organization_id = ?", orgName).Error
	return &server, err
}

// GetAvailableServer returns an available server
func GetAvailableServer(db *gorm.DB) (*Server, error) {
	var server Server
	err := db.Find(&server, "organization_id is null").Error
	return &server, err
}
