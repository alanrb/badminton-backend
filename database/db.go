package database

import (
	"fmt"

	"github.com/alanrb/badminton/backend/models"
	"github.com/alanrb/badminton/backend/rbac"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TransactionHandler defines a function type for transaction operations
type TransactionHandler func(tx *gorm.DB) error

var DB *gorm.DB

func Init(host, port, user, password, db_name, ssl_mode string, debug bool) {
	dsn := "host=" + host +
		" port=" + port +
		" user=" + user +
		" password=" + password +
		" dbname=" + db_name +
		" sslmode=" + ssl_mode

	if debug {
		fmt.Printf("connect to: %v\n", dsn)
	}
	var err error

	var lg logger.Interface
	if debug {
		lg = logger.Default.LogMode(logger.LogLevel(4))
	} else {
		lg = logger.Discard
	}

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   lg,
		DisableForeignKeyConstraintWhenMigrating: true,
	})

	if err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}
	if err := DB.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.SessionAttendee{},
		&models.BadmintonCourt{},
		&models.Group{},
		&models.GroupMember{},
		&models.GroupSession{},
		&models.Role{},
		&models.Permission{},
		&models.UserRole{},
	); err != nil {
		panic(fmt.Sprintf("Failed to migrate database: %v", err))
	}

	rbac.SeedRolesAndPermissions(DB)
}

// RunInTransaction runs the provided handler function within a database transaction
func RunInTransaction(db *gorm.DB, handler TransactionHandler) error {
	// Start a new transaction
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Defer a function to handle rollback in case of errors
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Execute the handler function within the transaction
	if err := handler(tx); err != nil {
		tx.Rollback()
		return err
	}

	// Commit the transaction if everything succeeds
	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}
