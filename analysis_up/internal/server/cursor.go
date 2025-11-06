package server

import (
	pdb "analysis/internal/db"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GET /sync/cursor?entity=binance&chain=ethereum
func GetCursor(gdb *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		entity := strings.TrimSpace(c.Query("entity"))
		chain := strings.TrimSpace(c.Query("chain"))
		if entity == "" || chain == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "entity and chain are required"})
			return
		}
		block, err := pdb.GetCursor(gdb, entity, chain)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"block": block})
	}
}

// POST /sync/cursor?entity=binance&chain=ethereum   body: {"block": 12345678}
func SetCursor(gdb *gorm.DB) gin.HandlerFunc {
	type req struct {
		Block uint64 `json:"block"`
	}
	return func(c *gin.Context) {
		entity := strings.TrimSpace(c.Query("entity"))
		chain := strings.TrimSpace(c.Query("chain"))
		if entity == "" || chain == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "entity and chain are required"})
			return
		}
		var body req
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}
		if body.Block == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "block must be > 0"})
			return
		}
		if err := pdb.UpsertCursor(gdb, entity, chain, body.Block); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "block": strconv.FormatUint(body.Block, 10)})
	}
}
