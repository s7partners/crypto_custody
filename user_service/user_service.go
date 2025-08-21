package user_service

import (
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	_ "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"net/http"
)

// ----------------- 数据库模型 -----------------
type User struct {
	ID        uint64 `gorm:"primaryKey"`
	Email     string `gorm:"uniqueIndex"`
	Phone     string `gorm:"uniqueIndex"`
	Password  string
	CreatedAt int64
	UpdatedAt int64
}

type KYC struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64
	Name      string
	IDNumber  string
	Status    string // pending/approved/rejected
	CreatedAt int64
	UpdatedAt int64
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

// ----------------- REST API -----------------
func registerUser(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := c.MustGet("db").(*gorm.DB)
	user := User{
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password, // 生产环境请加密存储
	}

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// TODO: 发布 UserRegistered MQ 事件
	c.JSON(http.StatusOK, gin.H{"userId": user.ID, "status": "created"})
}

func loginUser(c *gin.Context) {
	var req struct {
		EmailOrPhone string `json:"email_or_phone"`
		Password     string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := c.MustGet("db").(*gorm.DB)
	var user User
	if err := db.Where("email=? OR phone=?", req.EmailOrPhone, req.EmailOrPhone).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	if user.Password != req.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		return
	}

	// TODO: 生成 JWT token
	c.JSON(http.StatusOK, gin.H{"userId": user.ID, "token": "jwt_token_placeholder"})
}

func submitKYC(c *gin.Context) {
	var req struct {
		UserID   uint64 `json:"userId"`
		Name     string `json:"name"`
		IDNumber string `json:"idNumber"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := c.MustGet("db").(*gorm.DB)
	kyc := KYC{
		UserID:   req.UserID,
		Name:     req.Name,
		IDNumber: req.IDNumber,
		Status:   "pending",
	}
	if err := db.Create(&kyc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// TODO: 发布 KYCSubmitted MQ 事件
	c.JSON(http.StatusOK, gin.H{"kycId": kyc.ID, "status": kyc.Status})
}

// ----------------- 中间件 -----------------
func dbMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	}
}

// ----------------- 启动服务 -----------------
func main() {
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
