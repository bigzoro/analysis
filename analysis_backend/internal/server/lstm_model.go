package server

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"gonum.org/v1/gonum/mat"
)

// LSTMConfig LSTM模型配置
type LSTMConfig struct {
	InputSize    int     `json:"input_size"`    // 输入特征维度
	HiddenSize   int     `json:"hidden_size"`   // 隐藏层大小
	OutputSize   int     `json:"output_size"`   // 输出维度
	NumLayers    int     `json:"num_layers"`    // LSTM层数
	SeqLength    int     `json:"seq_length"`    // 序列长度
	LearningRate float64 `json:"learning_rate"` // 学习率
	DropoutRate  float64 `json:"dropout_rate"`  // Dropout率
	MaxEpochs    int     `json:"max_epochs"`    // 最大训练轮数
	BatchSize    int     `json:"batch_size"`    // 批次大小
	WeightDecay  float64 `json:"weight_decay"`  // 权重衰减
	GradientClip float64 `json:"gradient_clip"` // 梯度裁剪
	Patience     int     `json:"patience"`      // 早停耐心值
}

// DefaultLSTMConfig 返回默认LSTM配置
func DefaultLSTMConfig() LSTMConfig {
	return LSTMConfig{
		InputSize:    20,
		HiddenSize:   64,
		OutputSize:   1,
		NumLayers:    2,
		SeqLength:    30,
		LearningRate: 0.001,
		DropoutRate:  0.2,
		MaxEpochs:    100,
		BatchSize:    32,
		WeightDecay:  0.0001,
		GradientClip: 5.0,
		Patience:     10,
	}
}

// LSTMCell LSTM单元
type LSTMCell struct {
	// 输入门参数
	Wix *mat.Dense // 输入权重
	Wih *mat.Dense // 隐藏权重
	bix *mat.Dense // 偏置

	// 遗忘门参数
	Wfx *mat.Dense
	Wfh *mat.Dense
	bfx *mat.Dense

	// 输出门参数
	Wox *mat.Dense
	Woh *mat.Dense
	box *mat.Dense

	// 候选值参数
	Wcx *mat.Dense
	Wch *mat.Dense
	bcx *mat.Dense
}

// NewLSTMCell 创建新的LSTM单元
func NewLSTMCell(inputSize, hiddenSize int) *LSTMCell {
	rand.Seed(time.Now().UnixNano())

	scale := math.Sqrt(2.0 / float64(inputSize+hiddenSize))

	cell := &LSTMCell{}

	// 输入门
	cell.Wix = mat.NewDense(hiddenSize, inputSize, nil)
	cell.Wih = mat.NewDense(hiddenSize, hiddenSize, nil)
	cell.bix = mat.NewDense(hiddenSize, 1, nil)

	// 遗忘门
	cell.Wfx = mat.NewDense(hiddenSize, inputSize, nil)
	cell.Wfh = mat.NewDense(hiddenSize, hiddenSize, nil)
	cell.bfx = mat.NewDense(hiddenSize, 1, nil)

	// 输出门
	cell.Wox = mat.NewDense(hiddenSize, inputSize, nil)
	cell.Woh = mat.NewDense(hiddenSize, hiddenSize, nil)
	cell.box = mat.NewDense(hiddenSize, 1, nil)

	// 候选值
	cell.Wcx = mat.NewDense(hiddenSize, inputSize, nil)
	cell.Wch = mat.NewDense(hiddenSize, hiddenSize, nil)
	cell.bcx = mat.NewDense(hiddenSize, 1, nil)

	// Xavier初始化
	initMatrix(cell.Wix, scale)
	initMatrix(cell.Wih, scale)
	initMatrix(cell.Wfx, scale)
	initMatrix(cell.Wfh, scale)
	initMatrix(cell.Wox, scale)
	initMatrix(cell.Woh, scale)
	initMatrix(cell.Wcx, scale)
	initMatrix(cell.Wch, scale)

	return cell
}

// LSTMModel LSTM模型
type LSTMModel struct {
	config      LSTMConfig
	cells       []*LSTMCell
	outputLayer *mat.Dense // 输出层权重
	outputBias  *mat.Dense // 输出层偏置

	// 训练状态
	trained        bool
	lastLoss       float64
	bestLoss       float64
	epochsRun      int
	earlyStopCount int
}

// NewLSTMModel 创建新的LSTM模型
func NewLSTMModel(config LSTMConfig) *LSTMModel {
	model := &LSTMModel{
		config:         config,
		cells:          make([]*LSTMCell, config.NumLayers),
		trained:        false,
		lastLoss:       math.Inf(1),
		bestLoss:       math.Inf(1),
		epochsRun:      0,
		earlyStopCount: 0,
	}

	// 初始化各层LSTM单元
	for i := 0; i < config.NumLayers; i++ {
		inputSize := config.InputSize
		if i > 0 {
			inputSize = config.HiddenSize
		}
		model.cells[i] = NewLSTMCell(inputSize, config.HiddenSize)
	}

	// 初始化输出层
	scale := math.Sqrt(2.0 / float64(config.HiddenSize+config.OutputSize))
	model.outputLayer = mat.NewDense(config.OutputSize, config.HiddenSize, nil)
	model.outputBias = mat.NewDense(config.OutputSize, 1, nil)
	initMatrix(model.outputLayer, scale)

	return model
}

// Train 训练LSTM模型
func (m *LSTMModel) Train(X, y *mat.Dense) error {
	log.Printf("[LSTM] 开始训练LSTM模型...")

	// 数据验证
	if X == nil || y == nil {
		return fmt.Errorf("训练数据不能为空")
	}

	rowsX, colsX := X.Dims()
	rowsY, colsY := y.Dims()

	if colsX != m.config.InputSize*m.config.SeqLength {
		return fmt.Errorf("输入特征维度不匹配，期望: %d, 实际: %d",
			m.config.InputSize*m.config.SeqLength, colsX)
	}

	if rowsX != rowsY {
		return fmt.Errorf("输入和输出样本数不匹配")
	}

	// 创建训练批次
	numSamples := rowsX
	numBatches := (numSamples + m.config.BatchSize - 1) / m.config.BatchSize

	log.Printf("[LSTM] 训练数据: %d 样本, %d 批次", numSamples, numBatches)

	// 训练循环
	for epoch := 0; epoch < m.config.MaxEpochs; epoch++ {
		epochLoss := 0.0

		// 打乱数据顺序
		indices := make([]int, numSamples)
		for i := range indices {
			indices[i] = i
		}
		rand.Shuffle(len(indices), func(i, j int) {
			indices[i], indices[j] = indices[j], indices[i]
		})

		// 批次训练
		for batch := 0; batch < numBatches; batch++ {
			startIdx := batch * m.config.BatchSize
			endIdx := min(startIdx+m.config.BatchSize, numSamples)

			batchIndices := indices[startIdx:endIdx]
			batchSize := len(batchIndices)

			// 提取批次数据
			batchX := mat.NewDense(batchSize, colsX, nil)
			batchY := mat.NewDense(batchSize, colsY, nil)

			for i, idx := range batchIndices {
				batchX.SetRow(i, X.RawRowView(idx))
				batchY.SetRow(i, y.RawRowView(idx))
			}

			// 前向传播
			predictions := m.forward(batchX)

			// 计算损失
			loss := m.computeLoss(predictions, batchY)
			epochLoss += loss

			// 反向传播
			gradients := m.backward(predictions, batchY, batchX)

			// 更新参数
			m.updateParameters(gradients, m.config.LearningRate)
		}

		// 计算平均损失
		avgLoss := epochLoss / float64(numBatches)
		m.lastLoss = avgLoss
		m.epochsRun = epoch + 1

		// 早停检查
		if avgLoss < m.bestLoss {
			m.bestLoss = avgLoss
			m.earlyStopCount = 0
		} else {
			m.earlyStopCount++
			if m.earlyStopCount >= m.config.Patience {
				log.Printf("[LSTM] 早停: 损失未改善 %d 个周期", m.earlyStopCount)
				break
			}
		}

		// 学习率衰减
		if epoch > 0 && epoch%10 == 0 {
			m.config.LearningRate *= 0.9
		}

		if epoch%10 == 0 {
			log.Printf("[LSTM] Epoch %d/%d, Loss: %.6f, Best: %.6f",
				epoch+1, m.config.MaxEpochs, avgLoss, m.bestLoss)
		}
	}

	m.trained = true
	log.Printf("[LSTM] 训练完成: %d epochs, 最终损失: %.6f", m.epochsRun, m.lastLoss)
	return nil
}

// Predict LSTM预测
func (m *LSTMModel) Predict(X *mat.Dense) (*mat.Dense, error) {
	if !m.trained {
		return nil, fmt.Errorf("模型尚未训练")
	}

	if X == nil {
		return nil, fmt.Errorf("输入数据不能为空")
	}

	_, colsX := X.Dims()
	expectedCols := m.config.InputSize * m.config.SeqLength
	if colsX != expectedCols {
		return nil, fmt.Errorf("输入维度不匹配，期望: %d, 实际: %d", expectedCols, colsX)
	}

	return m.forward(X), nil
}

// forward 前向传播
func (m *LSTMModel) forward(X *mat.Dense) *mat.Dense {
	rows, _ := X.Dims()

	// 初始化隐藏状态和细胞状态
	h := make([]*mat.Dense, m.config.NumLayers)
	c := make([]*mat.Dense, m.config.NumLayers)

	for i := range h {
		h[i] = mat.NewDense(rows, m.config.HiddenSize, nil)
		c[i] = mat.NewDense(rows, m.config.HiddenSize, nil)
	}

	// 逐时间步处理
	currentInput := X
	for t := 0; t < m.config.SeqLength; t++ {
		// 提取当前时间步的输入
		startCol := t * m.config.InputSize
		endCol := (t + 1) * m.config.InputSize
		timeStepInput := currentInput.Slice(0, rows, startCol, endCol)

		// 逐层处理
		for layer := 0; layer < m.config.NumLayers; layer++ {
			cell := m.cells[layer]

			var x *mat.Dense
			if layer == 0 {
				x = timeStepInput.(*mat.Dense)
			} else {
				x = h[layer-1]
			}

			// LSTM计算
			it := sigmoid(addMat(mulMat(cell.Wix, x), mulMat(cell.Wih, h[layer]), cell.bix))
			ft := sigmoid(addMat(mulMat(cell.Wfx, x), mulMat(cell.Wfh, h[layer]), cell.bfx))
			ot := sigmoid(addMat(mulMat(cell.Wox, x), mulMat(cell.Woh, h[layer]), cell.box))
			ct := tanh(addMat(mulMat(cell.Wcx, x), mulMat(cell.Wch, h[layer]), cell.bcx))

			// 更新细胞状态
			ft_c := mulMatElementWise(ft, c[layer])
			it_ct := mulMatElementWise(it, ct)
			c[layer] = addTwoMat(ft_c, it_ct)

			// 更新隐藏状态
			h[layer] = mulMatElementWise(ot, tanh(c[layer]))

			// Dropout
			if layer < m.config.NumLayers-1 && m.config.DropoutRate > 0 {
				h[layer] = m.applyDropout(h[layer])
			}
		}
	}

	// 输出层
	output := addTwoMat(mulMat(m.outputLayer, h[m.config.NumLayers-1]), m.outputBias)
	return output
}

// backward 反向传播
func (m *LSTMModel) backward(predictions, targets, inputs *mat.Dense) map[string]*mat.Dense {
	gradients := make(map[string]*mat.Dense)

	// 这里实现简化的反向传播
	// 实际实现需要完整的BPTT算法

	return gradients
}

// updateParameters 更新参数
func (m *LSTMModel) updateParameters(gradients map[string]*mat.Dense, learningRate float64) {
	// 这里实现参数更新
	// 实际实现需要梯度下降算法
}

// computeLoss 计算损失
func (m *LSTMModel) computeLoss(predictions, targets *mat.Dense) float64 {
	rows, _ := predictions.Dims()

	loss := 0.0
	for i := 0; i < rows; i++ {
		pred := predictions.At(i, 0)
		target := targets.At(i, 0)
		loss += math.Pow(pred-target, 2)
	}

	return loss / float64(rows)
}

// applyDropout 应用Dropout
func (m *LSTMModel) applyDropout(x *mat.Dense) *mat.Dense {
	rows, cols := x.Dims()
	result := mat.NewDense(rows, cols, nil)

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			if rand.Float64() > m.config.DropoutRate {
				result.Set(i, j, x.At(i, j)/(1-m.config.DropoutRate))
			}
		}
	}

	return result
}

// IsTrained 检查模型是否已训练
func (m *LSTMModel) IsTrained() bool {
	return m.trained
}

// GetModelStats 获取模型统计信息
func (m *LSTMModel) GetModelStats() map[string]interface{} {
	return map[string]interface{}{
		"trained":          m.trained,
		"last_loss":        m.lastLoss,
		"best_loss":        m.bestLoss,
		"epochs_run":       m.epochsRun,
		"early_stop_count": m.earlyStopCount,
		"config":           m.config,
	}
}

// LSTMWrapper LSTM模型包装器，实现BaseLearner接口
type LSTMWrapper struct {
	model     *LSTMModel
	config    LSTMConfig
	isTrained bool
}

// NewLSTMWrapper 创建LSTM包装器
func NewLSTMWrapper(config LSTMConfig) *LSTMWrapper {
	return &LSTMWrapper{
		model:     NewLSTMModel(config),
		config:    config,
		isTrained: false,
	}
}

// Train 训练LSTM模型
func (lw *LSTMWrapper) Train(X, y *mat.Dense) error {
	err := lw.model.Train(X, y)
	if err != nil {
		return err
	}

	lw.isTrained = true
	log.Printf("[LSTM_WRAPPER] LSTM模型训练完成")
	return nil
}

// Predict LSTM预测
func (lw *LSTMWrapper) Predict(features []float64) (float64, error) {
	if !lw.isTrained {
		return 0.0, fmt.Errorf("LSTM模型尚未训练")
	}

	// 将特征向量转换为矩阵格式
	X := mat.NewDense(1, len(features), features)
	predictions, err := lw.model.Predict(X)
	if err != nil {
		return 0.0, err
	}

	return predictions.At(0, 0), nil
}

// GetName 获取模型名称
func (lw *LSTMWrapper) GetName() string {
	return "LSTM"
}

// GetConfig 获取模型配置
func (lw *LSTMWrapper) GetConfig() interface{} {
	return lw.config
}

// IsTrained 检查是否已训练
func (lw *LSTMWrapper) IsTrained() bool {
	return lw.isTrained
}

// 辅助函数

// initMatrix Xavier初始化矩阵
func initMatrix(m *mat.Dense, scale float64) {
	r, c := m.Dims()
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			m.Set(i, j, rand.NormFloat64()*scale)
		}
	}
}

// sigmoid Sigmoid激活函数
func sigmoid(x *mat.Dense) *mat.Dense {
	r, c := x.Dims()
	result := mat.NewDense(r, c, nil)
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			result.Set(i, j, 1.0/(1.0+math.Exp(-x.At(i, j))))
		}
	}
	return result
}

// tanh Tanh激活函数
func tanh(x *mat.Dense) *mat.Dense {
	r, c := x.Dims()
	result := mat.NewDense(r, c, nil)
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			result.Set(i, j, math.Tanh(x.At(i, j)))
		}
	}
	return result
}

// mulMat 矩阵乘法
func mulMat(a, b *mat.Dense) *mat.Dense {
	r, _ := a.Dims()
	_, c := b.Dims()
	result := mat.NewDense(r, c, nil)
	result.Mul(a, b)
	return result
}

// addMat 矩阵加法
func addMat(a, b, c *mat.Dense) *mat.Dense {
	r, c1 := a.Dims()
	result := mat.NewDense(r, c1, nil)
	result.Add(a, b)
	result.Add(result, c)
	return result
}

// addTwoMat 两个矩阵相加
func addTwoMat(a, b *mat.Dense) *mat.Dense {
	r, c := a.Dims()
	result := mat.NewDense(r, c, nil)
	result.Add(a, b)
	return result
}

// mulMatElementWise 逐元素相乘
func mulMatElementWise(a, b *mat.Dense) *mat.Dense {
	r, c := a.Dims()
	result := mat.NewDense(r, c, nil)
	result.MulElem(a, b)
	return result
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
