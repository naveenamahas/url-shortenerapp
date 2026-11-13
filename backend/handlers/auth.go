package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"

	"urlshortener/models"
	"urlshortener/utils"
)

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) token(u models.User) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   u.ID.Hex(),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.Cfg.JWTSecret))
}

// POST /api/auth/signup
func (h *Handler) Signup(c *gin.Context) {
	var in credentials
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	email, err := utils.ValidateEmail(in.Email)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	// bcrypt only reads the first 72 bytes, so we cap there.
	if len(in.Password) < 8 || len(in.Password) > 72 {
		fail(c, http.StatusBadRequest, "password must be 8-72 characters")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(c, http.StatusInternalServerError, "could not create account")
		return
	}
	user := models.User{ID: primitive.NewObjectID(), Email: email, PasswordHash: string(hash), CreatedAt: time.Now().UTC()}
	if _, err := h.Users.InsertOne(c.Request.Context(), user); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			fail(c, http.StatusConflict, "an account with this email already exists")
			return
		}
		fail(c, http.StatusInternalServerError, "could not create account")
		return
	}
	tok, err := h.token(user)
	if err != nil {
		fail(c, http.StatusInternalServerError, "could not create token")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"token": tok, "user": user})
}

// POST /api/auth/login
func (h *Handler) Login(c *gin.Context) {
	var in credentials
	if err := c.ShouldBindJSON(&in); err != nil {
		fail(c, http.StatusBadRequest, "invalid request body")
		return
	}
	var user models.User
	err := h.Users.FindOne(c.Request.Context(), bson.M{"email": strings.ToLower(strings.TrimSpace(in.Email))}).Decode(&user)
	// Same message for "no such user" and "wrong password" so attackers
	// cannot discover which emails are registered.
	const bad = "incorrect email or password"
	if err != nil {
		fail(c, http.StatusUnauthorized, bad)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)) != nil {
		fail(c, http.StatusUnauthorized, bad)
		return
	}
	tok, err := h.token(user)
	if err != nil {
		fail(c, http.StatusInternalServerError, "could not create token")
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tok, "user": user})
}
