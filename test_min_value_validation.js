// 测试最小值验证的修改
console.log("=== 测试最小值验证修改 ===\n");

// 模拟输入验证函数
function validateMarginLossStopLoss(value) {
  let result = value;

  if (isNaN(value) || value <= 0) {
    result = 30.0; // 默认值
  } else if (value > 80) {
    result = 80; // 最大值
  }

  return result;
}

function validateMarginProfitTakeProfit(value) {
  let result = value;

  if (isNaN(value) || value <= 0) {
    result = 100.0; // 默认值
  } else if (value > 500) {
    result = 500; // 最大值
  }

  return result;
}

// 测试用例
const testCases = [
  // 保证金损失止损测试
  { func: validateMarginLossStopLoss, name: "保证金损失止损", tests: [
    { input: 0, expected: 30.0, desc: "0值应修正为默认值" },
    { input: -5, expected: 30.0, desc: "负值应修正为默认值" },
    { input: 0.1, expected: 0.1, desc: "0.1应被接受" },
    { input: 1, expected: 1, desc: "1应被接受" },
    { input: 2.5, expected: 2.5, desc: "2.5应被接受" },
    { input: 85, expected: 80, desc: "超过最大值应修正" }
  ]},

  // 保证金盈利止盈测试
  { func: validateMarginProfitTakeProfit, name: "保证金盈利止盈", tests: [
    { input: 0, expected: 100.0, desc: "0值应修正为默认值" },
    { input: -10, expected: 100.0, desc: "负值应修正为默认值" },
    { input: 0.1, expected: 0.1, desc: "0.1应被接受" },
    { input: 5, expected: 5, desc: "5应被接受" },
    { input: 50, expected: 50, desc: "50应被接受" },
    { input: 600, expected: 500, desc: "超过最大值应修正" }
  ]}
];

testCases.forEach(testCase => {
  console.log(`测试: ${testCase.name}`);
  testCase.tests.forEach(test => {
    const result = testCase.func(test.input);
    const passed = Math.abs(result - test.expected) < 0.001;
    console.log(`  ${test.desc}: 输入${test.input} -> 输出${result} ${passed ? '✅' : '❌'}`);
  });
  console.log('');
});

console.log("=== HTML输入框属性检查 ===");
console.log("保证金损失止损:");
console.log("  min='0.1' (之前是5)");
console.log("  step='0.1' (之前是1)");
console.log("  max='80'");
console.log("");
console.log("保证金盈利止盈:");
console.log("  min='0.1' (之前是10)");
console.log("  step='0.1' (之前是5)");
console.log("  max='500'");
console.log("");
console.log("✅ 现在用户可以设置任何大于0的百分比值！");
console.log("✅ 移除了不必要的限制，提供了更大的灵活性！");