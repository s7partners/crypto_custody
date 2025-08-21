package main

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func main() { //TIP <p>Press <shortcut actionId="ShowIntentionActions"/> when your caret is at the underlined text
	// to see how GoLand suggests fixing the warning.</p><p>Alternatively, if available, click the lightbulb to view possible fixes.</p>
	db := initDB()
	r := gin.Default()
	r.Use(dbMiddleware(db))

	api := r.Group("/api/v1/user")
	{
		api.POST("/register", registerUser)
		api.POST("/login", loginUser)
		api.POST("/kyc/submit", submitKYC)
		// TODO: /kyc/status, /withdraw/apply, /balance 等接口
	}

	log.Println("User Service running on :8080")
	r.Run(":8080")
}

// ----------------- 初始化数据库 -----------------
func initDB() *gorm.DB {
	dsn := "host=localhost user=postgres password=postgres dbname=userservice port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// 自动迁移
	db.AutoMigrate(&User{}, &KYC{})
	return db
}
