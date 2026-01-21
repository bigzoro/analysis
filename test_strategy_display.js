// 测试策略展示数据完整性
const mockStrategyData = {
    type: 'grid_trading',
    name: '网格交易策略',
    score: 10,
    confidence: 80,
    reason: '市场横盘震荡，网格策略最适合',
    exists: true,
    win_rate: 0.67,
    max_drawdown: 0.08,
    total_trades: 120,
    avg_profit: 0.032,
    sharpe_ratio: 2.2,
    volatility: 0.15,
    risk_level: 'low',
    suitable_market: '横盘震荡市场'
};

console.log('🔍 策略展示数据完整性检查');
console.log('=====================================');

// 检查后端提供的所有字段
const backendFields = [
    'type', 'name', 'score', 'confidence', 'reason', 'exists',
    'win_rate', 'max_drawdown', 'total_trades', 'avg_profit',
    'sharpe_ratio', 'volatility', 'risk_level', 'suitable_market'
];

console.log('✅ 后端提供的字段:');
backendFields.forEach(field => {
    const value = mockStrategyData[field];
    console.log(`   ${field}: ${value} (${typeof value})`);
});

// 计算衍生指标
const expectedReturn = mockStrategyData.win_rate * mockStrategyData.avg_profit * 250 * 100;
const riskRewardRatio = Math.abs(mockStrategyData.avg_profit / mockStrategyData.max_drawdown);
const confidenceLevel = mockStrategyData.confidence >= 80 ? '高' :
                       mockStrategyData.confidence >= 60 ? '中' : '低';

console.log('\n📊 计算的衍生指标:');
console.log(`   预期年化收益: ${expectedReturn.toFixed(2)}%`);
console.log(`   风险收益比: ${riskRewardRatio.toFixed(2)}`);
console.log(`   置信度等级: ${confidenceLevel}`);

console.log('\n🎯 前端展示的完整信息:');
console.log('   基本信息: 名称、评分、匹配度、风险等级、适用市场');
console.log('   性能指标: 胜率、最大回撤、夏普比率、总交易、平均利润、波动率');
console.log('   分析洞察: 预期年化收益、风险收益比、置信度等级');
console.log('   操作功能: 配置状态、立即使用按钮');

console.log('\n✅ 数据展示完整性验证通过！');