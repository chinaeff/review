package main

import (
	"database/sql"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

var (
	db           *sql.DB
	usersCache   map[string]User
	usersCacheMu sync.Mutex
)

type User struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Age      int    `json:"age"`
}

func main() {
	db, err := sql.Open("sqlite3", "users.db")
	if err != nil {
		return
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			email TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			name TEXT NOT NULL,
			age INTEGER NOT NULL
		)
	`)
	if err != nil {
		return
	}

	router := gin.Default()

	router.POST("/register", registerHandler)
	router.GET("/users", getUsersHandler)

	err = router.Run(":8080")
	if err != nil {
		return
	}
}

func registerHandler(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	usersCacheMu.Lock()
	defer usersCacheMu.Unlock()

	if existingUser, ok := usersCache[user.Email]; ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "", "existing": existingUser})
		return
	}

	if user.Age < 18 {
		c.JSON(http.StatusBadRequest, gin.H{"error": ""})
		return
	}

	_, err := db.Exec("INSERT INTO users (email, password, name, age) VALUES (?, ?, ?, ?)", user.Email,
		user.Password, user.Name, user.Age)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": ""})
		return
	}

	usersCache[user.Email] = user

	c.JSON(http.StatusOK, gin.H{"message": "User registered", "user": struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Age   int    `json:"age"`
	}{
		Email: user.Email,
		Name:  user.Name,
		Age:   user.Age,
	}})
}

func getUsersHandler(c *gin.Context) {
	usersCacheMu.Lock()
	defer usersCacheMu.Unlock()

	rows, err := db.Query("SELECT email, name, age FROM users")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": ""})
		return
	}
	defer rows.Close()

	var users []struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Age   int    `json:"age"`
	}
	for rows.Next() {
		var user struct {
			Email string `json:"email"`
			Name  string `json:"name"`
			Age   int    `json:"age"`
		}
		if err := rows.Scan(&user.Email, &user.Name, &user.Age); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed"})
			return
		}
		users = append(users, user)
	}

	c.JSON(http.StatusOK, users)
}

