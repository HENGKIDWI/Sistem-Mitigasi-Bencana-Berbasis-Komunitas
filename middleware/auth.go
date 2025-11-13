// middleware/auth.go
package middleware

import (
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// AuthMiddleware verifies JWT token
func AuthMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   true,
			"message": "Missing authorization header",
		})
	}

	// Extract token from "Bearer <token>"
	tokenString := strings.Replace(authHeader, "Bearer ", "", 1)

	// Parse and validate token
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid or expired token",
		})
	}

	// Extract claims
	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid token claims",
		})
	}

	// Store user info in context
	c.Locals("userID", claims.UserID)
	c.Locals("role", claims.Role)

	return c.Next()
}

// RoleMiddleware checks if user has required role
func RoleMiddleware(allowedRoles []string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("role").(string)

		// Check if user role is in allowed roles
		for _, role := range allowedRoles {
			if role == userRole {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   true,
			"message": "You don't have permission to access this resource",
		})
	}
}

// GenerateToken creates JWT token
func GenerateToken(userID uint, role string) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
