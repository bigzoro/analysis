package db

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// CreateTradingStrategy 创建策略
func CreateTradingStrategy(gdb *gorm.DB, strategy *TradingStrategy) error {
	if strategy.UserID == 0 {
		return fmt.Errorf("用户ID不能为空")
	}
	if strategy.Name == "" {
		return fmt.Errorf("策略名称不能为空")
	}

	strategy.CreatedAt = time.Now()
	strategy.UpdatedAt = time.Now()

	return gdb.Create(strategy).Error
}

// UpdateTradingStrategy 更新策略
func UpdateTradingStrategy(gdb *gorm.DB, strategy *TradingStrategy) error {
	if strategy.ID == 0 {
		return fmt.Errorf("策略ID不能为空")
	}

	strategy.UpdatedAt = time.Now()
	return gdb.Save(strategy).Error
}

// DeleteTradingStrategy 删除策略
func DeleteTradingStrategy(gdb *gorm.DB, userID, strategyID uint) error {
	result := gdb.Where("user_id = ? AND id = ?", userID, strategyID).Delete(&TradingStrategy{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetTradingStrategy 获取单个策略
func GetTradingStrategy(gdb *gorm.DB, userID, strategyID uint) (*TradingStrategy, error) {
	var strategy TradingStrategy
	err := gdb.Where("user_id = ? AND id = ?", userID, strategyID).First(&strategy).Error
	if err != nil {
		return nil, err
	}
	return &strategy, nil
}

// ListTradingStrategies 获取用户的所有策略
func ListTradingStrategies(gdb *gorm.DB, userID uint) ([]TradingStrategy, error) {
	var strategies []TradingStrategy
	err := gdb.Where("user_id = ?", userID).Order("created_at DESC").Find(&strategies).Error
	return strategies, err
}

// GetTradingStrategiesByIDs 根据ID列表获取策略（用于批量操作）
func GetTradingStrategiesByIDs(gdb *gorm.DB, userID uint, strategyIDs []uint) ([]TradingStrategy, error) {
	var strategies []TradingStrategy
	err := gdb.Where("user_id = ? AND id IN ?", userID, strategyIDs).Find(&strategies).Error
	return strategies, err
}

// ===== 策略执行相关 =====

// StartStrategyExecution 开始策略执行
func StartStrategyExecution(gdb *gorm.DB, execution *StrategyExecution) error {
	if execution.StrategyID == 0 || execution.UserID == 0 {
		return fmt.Errorf("策略ID和用户ID不能为空")
	}

	// 初始状态为pending，等待调度器开始执行
	execution.Status = "pending"
	execution.StartTime = time.Now()
	execution.CreatedAt = time.Now()
	execution.UpdatedAt = time.Now()

	return gdb.Create(execution).Error
}

// UpdateStrategyExecution 更新策略执行状态
func UpdateStrategyExecution(gdb *gorm.DB, execution *StrategyExecution) error {
	if execution.ID == 0 {
		return fmt.Errorf("执行ID不能为空")
	}

	execution.UpdatedAt = time.Now()
	return gdb.Save(execution).Error
}

// CompleteStrategyExecution 完成策略执行
func CompleteStrategyExecution(gdb *gorm.DB, executionID uint, totalOrders, successOrders, failedOrders int, totalPnL float64, logs string) error {
	var winRate float64
	if totalOrders > 0 {
		winRate = float64(successOrders) / float64(totalOrders) * 100
	}

	updates := map[string]interface{}{
		"status":         "completed",
		"end_time":       time.Now(),
		"total_orders":   totalOrders,
		"success_orders": successOrders,
		"failed_orders":  failedOrders,
		"total_pnl":      totalPnL,
		"win_rate":       winRate,
		"logs":           logs,
		"updated_at":     time.Now(),
	}

	// 计算执行时长
	var execution StrategyExecution
	if err := gdb.First(&execution, executionID).Error; err != nil {
		return err
	}
	if !execution.StartTime.IsZero() {
		duration := time.Since(execution.StartTime).Seconds()
		updates["duration"] = int(duration)
	}

	return gdb.Model(&StrategyExecution{}).Where("id = ?", executionID).Updates(updates).Error
}

// GetStrategyExecution 获取单个策略执行记录
func GetStrategyExecution(gdb *gorm.DB, userID, executionID uint) (*StrategyExecution, error) {
	var execution StrategyExecution
	err := gdb.Preload("Strategy").Where("user_id = ? AND id = ?", userID, executionID).First(&execution).Error
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

// ListStrategyExecutions 获取策略的执行记录列表
func ListStrategyExecutions(gdb *gorm.DB, userID, strategyID uint, limit int) ([]StrategyExecution, error) {
	var executions []StrategyExecution
	query := gdb.Preload("Strategy").Where("user_id = ?", userID)

	if strategyID > 0 {
		query = query.Where("strategy_id = ?", strategyID)
	}

	err := query.Order("created_at DESC").Limit(limit).Find(&executions).Error
	return executions, err
}

// ListStrategyExecutionsPaged 分页获取策略执行记录
func ListStrategyExecutionsPaged(gdb *gorm.DB, userID, strategyID uint, page, pageSize int) ([]StrategyExecution, int64, error) {
	var executions []StrategyExecution
	var total int64

	query := gdb.Model(&StrategyExecution{}).Where("user_id = ?", userID)

	if strategyID > 0 {
		query = query.Where("strategy_id = ?", strategyID)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	err := gdb.Preload("Strategy").Where("user_id = ?", userID).
		Where("strategy_id = ?", strategyID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&executions).Error

	return executions, total, err
}

// GetRunningStrategyExecutions 获取正在运行的策略执行
func GetRunningStrategyExecutions(gdb *gorm.DB, userID uint) ([]StrategyExecution, error) {
	var executions []StrategyExecution
	err := gdb.Preload("Strategy").Where("user_id = ? AND status = ?", userID, "running").Find(&executions).Error
	return executions, err
}

// StopStrategyExecution 停止策略执行
func StopStrategyExecution(gdb *gorm.DB, executionID uint) error {
	return gdb.Model(&StrategyExecution{}).Where("id = ?", executionID).Updates(map[string]interface{}{
		"status":     "completed",
		"end_time":   time.Now(),
		"updated_at": time.Now(),
	}).Error
}

// UpdateStrategyRunningStatus 更新策略运行状态
func UpdateStrategyRunningStatus(gdb *gorm.DB, strategyID uint, isRunning bool) error {
	updates := map[string]interface{}{
		"is_running": isRunning,
		"updated_at": time.Now(),
	}

	if !isRunning {
		// 停止运行时，记录最后运行时间
		updates["last_run_at"] = time.Now()
	} else {
		// 开始运行时，清除最后运行时间，让调度器立即执行
		updates["last_run_at"] = nil
	}

	return gdb.Model(&TradingStrategy{}).Where("id = ?", strategyID).Updates(updates).Error
}

// 更新策略执行状态和进度
func UpdateStrategyExecutionStatus(gdb *gorm.DB, executionID uint, status, currentStep, currentSymbol string, stepProgress, totalProgress int, errorMessage string) error {
	updates := map[string]interface{}{
		"status":         status,
		"current_step":   currentStep,
		"current_symbol": currentSymbol,
		"step_progress":  stepProgress,
		"total_progress": totalProgress,
		"updated_at":     time.Now(),
	}

	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	if status == "running" {
		updates["start_time"] = time.Now()
	} else if status == "completed" || status == "failed" {
		updates["end_time"] = time.Now()
	}

	return gdb.Model(&StrategyExecution{}).Where("id = ?", executionID).Updates(updates).Error
}

// 添加执行日志
func AppendStrategyExecutionLog(gdb *gorm.DB, executionID uint, logEntry string) error {
	var execution StrategyExecution
	if err := gdb.Where("id = ?", executionID).First(&execution).Error; err != nil {
		return err
	}

	newLogs := execution.Logs
	if newLogs != "" {
		newLogs += "\n"
	}
	newLogs += fmt.Sprintf("[%s] %s", time.Now().Format("2006-01-02 15:04:05"), logEntry)

	return gdb.Model(&StrategyExecution{}).Where("id = ?", executionID).Update("logs", newLogs).Error
}

// 创建执行步骤记录
func CreateStrategyExecutionStep(gdb *gorm.DB, step *StrategyExecutionStep) error {
	now := time.Now()
	step.CreatedAt = now
	step.UpdatedAt = now

	// 如果StartTime是nil，设置为当前时间
	if step.StartTime == nil {
		step.StartTime = &now
	}

	return gdb.Create(step).Error
}

// 更新执行步骤状态
func UpdateStrategyExecutionStep(gdb *gorm.DB, stepID uint, status string, result, errorMessage, data string) error {
	updates := map[string]interface{}{
		"status":     status,
		"result":     result,
		"updated_at": time.Now(),
	}

	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	if data != "" {
		updates["data"] = data
	}

	if status == "completed" || status == "failed" {
		updates["end_time"] = time.Now()
	}

	return gdb.Model(&StrategyExecutionStep{}).Where("id = ?", stepID).Updates(updates).Error
}

// 获取执行步骤列表
func GetStrategyExecutionSteps(gdb *gorm.DB, executionID uint) ([]StrategyExecutionStep, error) {
	var steps []StrategyExecutionStep
	err := gdb.Where("execution_id = ?", executionID).Order("created_at asc").Find(&steps).Error
	return steps, err
}

// 计算执行持续时间
func UpdateStrategyExecutionDuration(gdb *gorm.DB, executionID uint) error {
	var execution StrategyExecution
	if err := gdb.Where("id = ?", executionID).First(&execution).Error; err != nil {
		return err
	}

	if execution.EndTime != nil && !execution.StartTime.IsZero() {
		duration := int(execution.EndTime.Sub(execution.StartTime).Seconds())
		return gdb.Model(&StrategyExecution{}).Where("id = ?", executionID).Update("duration", duration).Error
	}

	return nil
}

// 获取所有正在运行的策略
func GetRunningStrategies(gdb *gorm.DB) ([]*TradingStrategy, error) {
	var strategies []*TradingStrategy
	err := gdb.Where("is_running = ?", true).Find(&strategies).Error
	return strategies, err
}

// 更新策略执行结果
func UpdateStrategyExecutionResult(gdb *gorm.DB, executionID uint, totalOrders, successOrders, failedOrders int, totalPnL, winRate float64) error {
	// 首先获取执行记录以计算持续时间
	var execution StrategyExecution
	if err := gdb.Where("id = ?", executionID).First(&execution).Error; err != nil {
		return err
	}

	endTime := time.Now()
	duration := int(endTime.Sub(execution.StartTime).Seconds())

	updates := map[string]interface{}{
		"total_orders":   totalOrders,
		"success_orders": successOrders,
		"failed_orders":  failedOrders,
		"total_pnl":      totalPnL,
		"win_rate":       winRate,
		"end_time":       endTime,
		"duration":       duration,
		"updated_at":     time.Now(),
	}

	return gdb.Model(&StrategyExecution{}).Where("id = ?", executionID).Updates(updates).Error
}

// UpdateStrategyExecutionResultWithStats 更新策略执行结果（包含完整的统计信息）
func UpdateStrategyExecutionResultWithStats(gdb *gorm.DB, executionID uint, totalOrders, successOrders, failedOrders int, totalPnL, winRate, pnlPercentage, totalInvestment, currentValue float64) error {
	// 首先获取执行记录以计算持续时间
	var execution StrategyExecution
	if err := gdb.Where("id = ?", executionID).First(&execution).Error; err != nil {
		return err
	}

	endTime := time.Now()
	duration := int(endTime.Sub(execution.StartTime).Seconds())

	updates := map[string]interface{}{
		"total_orders":     totalOrders,
		"success_orders":   successOrders,
		"failed_orders":    failedOrders,
		"total_pnl":        totalPnL,
		"win_rate":         winRate,
		"pnl_percentage":   pnlPercentage,
		"total_investment": totalInvestment,
		"current_value":    currentValue,
		"end_time":         endTime,
		"duration":         duration,
		"updated_at":       time.Now(),
	}

	return gdb.Model(&StrategyExecution{}).Where("id = ?", executionID).Updates(updates).Error
}

// DeleteStrategyExecution 删除策略执行记录
func DeleteStrategyExecution(gdb *gorm.DB, userID, executionID uint) error {
	// 首先检查执行记录是否存在且属于该用户
	var execution StrategyExecution
	if err := gdb.Where("id = ? AND user_id = ?", executionID, userID).First(&execution).Error; err != nil {
		return err
	}

	// 开始事务
	tx := gdb.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 先删除相关的执行步骤记录
	if err := tx.Where("execution_id = ?", executionID).Delete(&StrategyExecutionStep{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 再删除执行记录
	if err := tx.Delete(&execution).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	return tx.Commit().Error
}
