package server

import (
	pdb "analysis/internal/db"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func jwtSecret() []byte {
	sec := os.Getenv("JWT_SECRET")
	if sec == "" {
		sec = "dev_secret_change_me"
	}
	return []byte(sec)
}

type jwtClaims struct {
	UID      uint   `json:"uid"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func (s *Server) issueToken(u *pdb.User) (string, error) {
	ttl := 30 * 24 * time.Hour
	claims := jwtClaims{
		UID:      u.ID,
		Username: u.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

func parseToken(tok string) (*jwtClaims, error) {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return nil, errors.New("empty token")
	}
	t, err := jwt.ParseWithClaims(tok, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret(), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := t.Claims.(*jwtClaims); ok && t.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

func bearerFrom(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	// 兼容 WebSocket query
	if q := c.Query("token"); q != "" {
		return q
	}
	return ""
}

/*** REST: /auth/register ***/
type authReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) Register(c *gin.Context) {
	var req authReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Username) < 3 || len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid username or password"})
		return
	}
	var exists int64
	if err := s.db.Model(&pdb.User{}).Where("username = ?", req.Username).Count(&exists).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if exists > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	u := &pdb.User{Username: req.Username, PasswordHash: string(hash)}
	if err := s.db.Create(u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	tok, _ := s.issueToken(u)
	c.JSON(http.StatusOK, gin.H{"token": tok, "user": gin.H{"id": u.ID, "username": u.Username}})
}

/*** REST: /auth/login ***/
func (s *Server) Login(c *gin.Context) {
	var req authReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	var u pdb.User
	if err := s.db.Where("username = ?", req.Username).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "wrong credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "wrong credentials"})
		return
	}
	tok, _ := s.issueToken(&u)
	c.JSON(http.StatusOK, gin.H{"token": tok, "user": gin.H{"id": u.ID, "username": u.Username}})
}

/*** REST: /me ***/
func (s *Server) Me(c *gin.Context) {
	uid, _ := c.Get("uid")
	username, _ := c.Get("username")
	c.JSON(http.StatusOK, gin.H{"id": uid, "username": username})
}

/*** Middleware ***/
func (s *Server) JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tok := bearerFrom(c)
		if tok == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims, err := parseToken(tok)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("uid", claims.UID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
