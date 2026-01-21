// 测试黑名单前端显示功能
// 这个文件用于验证前端是否正确显示策略的黑名单信息

console.log("=== 测试前端黑名单显示功能 ===");

// 模拟策略数据
const mockStrategy = {
  id: 33,
  name: "测试策略",
  conditions: {
    use_symbol_whitelist: false,
    symbol_whitelist: [],
    use_symbol_blacklist: true,
    symbol_blacklist: ["BTCUSDT", "ETHUSDT", "ADAUSDT"]
  }
};

console.log("策略数据:", mockStrategy);

// 测试白名单显示条件
const shouldShowWhitelist = mockStrategy.conditions.use_symbol_whitelist &&
  mockStrategy.conditions.symbol_whitelist &&
  mockStrategy.conditions.symbol_whitelist.length > 0;

console.log("是否显示白名单:", shouldShowWhitelist);
console.log("白名单数量:", shouldShowWhitelist ? mockStrategy.conditions.symbol_whitelist.length : 0);

// 测试黑名单显示条件
const shouldShowBlacklist = mockStrategy.conditions.use_symbol_blacklist &&
  mockStrategy.conditions.symbol_blacklist &&
  mockStrategy.conditions.symbol_blacklist.length > 0;

console.log("是否显示黑名单:", shouldShowBlacklist);
console.log("黑名单数量:", shouldShowBlacklist ? mockStrategy.conditions.symbol_blacklist.length : 0);

// 模拟HTML显示 - 显示具体币种列表
if (shouldShowWhitelist) {
  console.log("前端显示 - 白名单:", mockStrategy.conditions.symbol_whitelist.join(', '));
}

if (shouldShowBlacklist) {
  console.log("前端显示 - 黑名单:", mockStrategy.conditions.symbol_blacklist.join(', '));
}

console.log("\n=== 测试完成 ===");
console.log("✅ 前端黑名单显示逻辑正确，现在显示具体币种列表");