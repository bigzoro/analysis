package db

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"size:64;uniqueIndex;not null"`
	PasswordHash string `gorm:"size:255;not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// AutoExecuteSettings 用户自动执行设置
type AutoExecuteSettings struct {
	ID             uint    `gorm:"primaryKey"`
	UserID         uint    `gorm:"column:user_id;index"`
	Enabled        bool    `gorm:"column:enabled;default:false"`                         // 是否启用自动执行
	RiskLevel      string  `gorm:"column:risk_level;size:16;default:'medium'"`           // conservative/moderate/aggressive
	MaxPosition    float64 `gorm:"column:max_position;type:decimal(5,2);default:5.00"`   // 最大单次仓位(%)
	MinConfidence  float64 `gorm:"column:min_confidence;type:decimal(5,2);default:0.70"` // 最小置信度
	MaxDailyTrades int     `gorm:"column:max_daily_trades;default:5"`                    // 每日最大交易次数
	Symbols        string  `gorm:"column:symbols;type:text"`                             // 允许交易的币种，用逗号分隔
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// TableName 指定表名
func (AutoExecuteSettings) TableName() string {
	return "auto_execute_settings"
}

// GetOrCreateAutoExecuteSettings 获取或创建用户的自动执行设置
func GetOrCreateAutoExecuteSettings(gdb *gorm.DB, userID uint) (*AutoExecuteSettings, error) {
	var settings AutoExecuteSettings
	err := gdb.Where("user_id = ?", userID).First(&settings).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建默认设置
			settings = AutoExecuteSettings{
				UserID:         userID,
				Enabled:        false,
				RiskLevel:      "medium",
				MaxPosition:    5.0,
				MinConfidence:  0.7,
				MaxDailyTrades: 5,
				Symbols:        "BTC,ETH,ADA,SOL,DOT",
			}
			err = gdb.Create(&settings).Error
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &settings, nil
}

// UpdateAutoExecuteSettings 更新用户的自动执行设置
func UpdateAutoExecuteSettings(gdb *gorm.DB, userID uint, settings *AutoExecuteSettings) error {
	settings.UserID = userID
	return gdb.Where("user_id = ?", userID).Save(settings).Error
}
