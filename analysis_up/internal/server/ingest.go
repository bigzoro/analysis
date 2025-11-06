package server

import (
	pdb "analysis/internal/db"
	"analysis/internal/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// POST /ingest/events?entity=binance
// Body: []models.Event
func IngestEvents(gdb *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		entity := strings.TrimSpace(c.Query("entity"))
		var evs []models.Event
		if err := c.BindJSON(&evs); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
			return
		}
		runID := uuid.NewString()
		rows, err := pdb.SaveTransferEvents(gdb, runID, entity, evs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// 只广播新插入的记录
		BroadcastTransfers(entity, rows)
		c.JSON(http.StatusOK, gin.H{"ok": true, "saved": len(rows), "run_id": runID})
	}
}
