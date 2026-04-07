package db

import (
	"fmt"

	"github.com/shukisha/backend/internal/platform/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init initializes the database connection using environment variables
func Init() (*gorm.DB, error) {
    dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=require TimeZone=Africa/Nairobi",
        config.GetEnv("DB_HOST", "localhost"),
        config.GetEnv("DB_USER", "postgres"),
        config.GetEnv("DB_PASS", "postgres"),
        config.GetEnv("DB_NAME", "shukisha"),
        config.GetEnv("DB_PORT", "5432"),
    )

    var err error
    DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
    })

    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

	fmt.Println("Database connection established successfully")
	return DB, nil
}
