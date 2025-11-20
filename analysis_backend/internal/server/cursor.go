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
			// 优化：使用统一的错误处理
			ValidationErrorHelper(c, "entity/chain", "entity 和 chain 参数不能为空")
			return
		}
		block, err := pdb.GetCursor(gdb, entity, chain)
		if err != nil {
			// 优化：使用统一的错误处理
			DatabaseErrorHelper(c, "查询游标", err)
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
			// 优化：使用统一的错误处理
			ValidationErrorHelper(c, "entity/chain", "entity 和 chain 参数不能为空")
			return
		}
		var body req
		if err := c.BindJSON(&body); err != nil {
			// 优化：使用统一的错误处理
			JSONBindErrorHelper(c, err)
			return
		}
		if body.Block == 0 {
			// 优化：使用统一的错误处理
			ValidationErrorHelper(c, "block", "block 必须大于 0")
			return
		}
		if err := pdb.UpsertCursor(gdb, entity, chain, body.Block); err != nil {
			// 优化：使用统一的错误处理
			DatabaseErrorHelper(c, "更新游标", err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "block": strconv.FormatUint(body.Block, 10)})
	}
}
