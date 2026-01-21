// 测试 Vue 导入修复
const { ref, computed } = require('vue');

// 测试计算属性
const testData = ref([1, 2, 3, 4, 5]);
const sum = computed(() => testData.value.reduce((a, b) => a + b, 0));

console.log('Test passed: computed is working');
console.log('Sum:', sum.value);