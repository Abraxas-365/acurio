package candidateauth

import (
	"strings"

	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/gofiber/fiber/v2"
)

// Middleware validates candidate session tokens
func Middleware(tokenService *CandidateTokenService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "Missing authorization header")
		}

		// Extract token (format: "Bearer <token>")
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid authorization format")
		}

		token := parts[1]

		// Validate token using our wrapper
		claims, err := tokenService.ValidateCandidateToken(token)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid or expired token")
		}

		// Set candidate info in context
		c.Locals("candidate_id", claims.CandidateID)
		c.Locals("candidate_email", claims.Email)

		return c.Next()
	}
}

// GetCandidateID extracts candidate ID from context
func GetCandidateID(c *fiber.Ctx) (kernel.CandidateID, bool) {
	candidateID, ok := c.Locals("candidate_id").(kernel.CandidateID)
	return candidateID, ok
}

// GetCandidateEmail extracts candidate email from context
func GetCandidateEmail(c *fiber.Ctx) (kernel.Email, bool) {
	email, ok := c.Locals("candidate_email").(kernel.Email)
	return email, ok
}

