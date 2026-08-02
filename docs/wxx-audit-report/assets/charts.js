// 审核报告图表
(function() {
  var style = getComputedStyle(document.documentElement);
  var accent = style.getPropertyValue('--accent').trim();
  var accent2 = style.getPropertyValue('--accent2').trim();
  var ink = style.getPropertyValue('--ink').trim();
  var muted = style.getPropertyValue('--muted').trim();
  var rule = style.getPropertyValue('--rule').trim();
  var bg2 = style.getPropertyValue('--bg2').trim();

  // 图1：各角色功能完成度
  var chart1 = echarts.init(document.getElementById('chart-role-completion'), null, { renderer: 'svg' });
  chart1.setOption({
    animation: false,
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' }, appendToBody: true },
    legend: { data: ['已完成', '部分完成/骨架', '未实现'], bottom: 0, textStyle: { color: muted } },
    grid: { left: '3%', right: '4%', bottom: '15%', top: '8%', containLabel: true },
    xAxis: {
      type: 'category',
      data: ['学生', '辅导员', '教师', '教辅', '学生会', '学院管理员', '学校管理员', '系统管理员'],
      axisLabel: { color: ink, fontSize: 11, interval: 0 },
      axisLine: { lineStyle: { color: rule } }
    },
    yAxis: {
      type: 'value',
      name: '功能数',
      axisLabel: { color: muted },
      axisLine: { lineStyle: { color: rule } },
      splitLine: { lineStyle: { color: rule, type: 'dashed' } }
    },
    series: [
      { name: '已完成', type: 'bar', stack: 'total', data: [45, 21, 18, 12, 7, 8, 5, 4], itemStyle: { color: accent }, barWidth: '40%' },
      { name: '部分完成/骨架', type: 'bar', stack: 'total', data: [3, 0, 0, 0, 0, 0, 0, 0], itemStyle: { color: accent2 } },
      { name: '未实现', type: 'bar', stack: 'total', data: [2, 0, 0, 0, 0, 0, 0, 0], itemStyle: { color: muted + '80' } }
    ]
  });
  window.addEventListener('resize', function() { chart1.resize(); });

  // 图2：P0/P1/P2完成度饼图
  var chart2 = echarts.init(document.getElementById('chart-priority-completion'), null, { renderer: 'svg' });
  chart2.setOption({
    animation: false,
    tooltip: { trigger: 'item', appendToBody: true, formatter: '{b}: {c} ({d}%)' },
    legend: { bottom: 0, textStyle: { color: muted } },
    series: [{
      type: 'pie',
      radius: ['40%', '70%'],
      center: ['50%', '45%'],
      label: { color: ink, fontSize: 12 },
      data: [
        { value: 28, name: 'P0 基础功能（全部完成）', itemStyle: { color: accent } },
        { value: 52, name: 'P1 核心特色（全部完成）', itemStyle: { color: accent2 } },
        { value: 25, name: 'P2 扩展功能（已完成）', itemStyle: { color: accent + '99' } },
        { value: 5, name: 'P2/P3 待完善', itemStyle: { color: muted + '60' } }
      ]
    }]
  });
  window.addEventListener('resize', function() { chart2.resize(); });

  // 图3：技术架构实现度
  var chart3 = echarts.init(document.getElementById('chart-tech-impl'), null, { renderer: 'svg' });
  chart3.setOption({
    animation: false,
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' }, appendToBody: true },
    grid: { left: '3%', right: '8%', bottom: '8%', top: '8%', containLabel: true },
    xAxis: {
      type: 'value',
      max: 100,
      axisLabel: { color: muted, formatter: '{value}%' },
      axisLine: { lineStyle: { color: rule } },
      splitLine: { lineStyle: { color: rule, type: 'dashed' } }
    },
    yAxis: {
      type: 'category',
      data: ['数据库迁移(50个)', '能力授权(104项)', '后端服务层', '后端路由注册', '前端页面', '前端路由', 'AI/LLM集成', '兜底策略'],
      axisLabel: { color: ink, fontSize: 11 },
      axisLine: { lineStyle: { color: rule } }
    },
    series: [{
      type: 'bar',
      data: [100, 100, 98, 100, 96, 100, 90, 95],
      itemStyle: {
        color: function(params) {
          return params.value >= 95 ? accent : accent2;
        }
      },
      label: { show: true, position: 'right', color: ink, formatter: '{c}%' },
      barWidth: '50%'
    }]
  });
  window.addEventListener('resize', function() { chart3.resize(); });
})();
