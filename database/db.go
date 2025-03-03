package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

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
	// if err := DB.AutoMigrate(&models.User{}, &models.Session{}, &models.SessionAttendee{}, &models.BadmintonCourt{}); err != nil {
	// 	panic(fmt.Sprintf("Failed to migrate database: %v", err))
	// }
}
