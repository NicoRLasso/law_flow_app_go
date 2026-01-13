package main

import (
	"bufio"
	"fmt"
	"law_flow_app_go/config"
	"law_flow_app_go/db"
	"law_flow_app_go/models"
	"law_flow_app_go/services"
	"log"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	if err := db.Initialize(cfg.DBPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.AutoMigrate(&models.Firm{}, &models.User{}, &models.Session{}); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	reader := bufio.NewReader(os.Stdin)

	// Get user details
	fmt.Println("=== Create New User ===")
	fmt.Println()

	fmt.Print("Name: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	fmt.Print("Email: ")
	email, _ := reader.ReadString('\n')
	email = strings.TrimSpace(email)

	// Get password securely
	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatalf("Failed to read password: %v", err)
	}
	password := string(passwordBytes)
	fmt.Println() // New line after password input

	// Validate inputs
	if name == "" || email == "" || password == "" {
		log.Fatal("Name, email, and password are required")
	}

	if len(password) < 8 {
		log.Fatal("Password must be at least 8 characters long")
	}

	// Check if user already exists
	var existingUser models.User
	if err := db.DB.Where("email = ?", email).First(&existingUser).Error; err == nil {
		log.Fatalf("User with email %s already exists", email)
	}

	// Hash password
	hashedPassword, err := services.HashPassword(password)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// Create user (without firm initially)
	user := &models.User{
		Name:     name,
		Email:    email,
		Password: hashedPassword,
		FirmID:   nil, // User will set up firm on first login
		Role:     "staff",
		IsActive: true,
	}

	if err := db.DB.Create(user).Error; err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	fmt.Println()
	fmt.Println("✓ User created successfully!")
	fmt.Printf("  ID: %d\n", user.ID)
	fmt.Printf("  Name: %s\n", user.Name)
	fmt.Printf("  Email: %s\n", user.Email)
	fmt.Println()
	fmt.Println("The user can now log in at http://localhost:8080/login")
	fmt.Println("They will be prompted to set up their firm on first login.")
}
