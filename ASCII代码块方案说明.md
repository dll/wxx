# ASCII 代码块图形展示方案

**更新时间**: 2025年10月23日  
**状态**: ✅ 已实施

---

## 📊 方案对比

### 之前：PNG 图片方案
- ✅ 显示一致性好
- ✅ 所有编辑器通用
- ❌ 需要额外的图片文件
- ❌ 修改不便（需要重新生成）
- ❌ 文件体积大

### 现在：ASCII 代码块 + 等宽字体
- ✅ 直接在文档中展示
- ✅ 易于编辑和维护
- ✅ 文件体积小
- ✅ 使用 Sarasa Mono SC 完美对齐
- ⚠️ 需要配置编辑器字体

---

## 🎨 实施内容

### 已替换的图形（共6处）

1. **手机端垂直布局图** (`Step 4.2`)
2. **平板端水平布局图** (`Step 4.3`)
3. **布局对比图** (`Step 4.5`)
4. **分布式数据架构图** (`Step 5.1`)
5. **跨设备适配对比图** (`5.1 功能2`)
6. **数据同步效果示意图** (`5.1 功能3`)
7. **屏幕布局检测图** (`4.5.2`)

### 代码块格式

所有 ASCII 图形使用以下格式：

````markdown
```ascii
┌────────────────┐
│  内容文本       │
└────────────────┘
```
````

---

## ⚙️ Typora 配置方法

### 快速配置（3步完成）

**步骤1：打开主题文件夹**
```
Typora → 文件 → 偏好设置 → 外观 → 打开主题文件夹
```

**步骤2：编辑 CSS 文件**

找到当前使用的主题文件（如 `github.css`），用文本编辑器打开，添加：

```css
/* ASCII 图形使用 Sarasa Mono SC 等宽字体 */
pre, 
code, 
.md-fences {
    font-family: 'Sarasa Mono SC', 'Consolas', 'Courier New', monospace !important;
    font-size: 13px;
    line-height: 1.6;
}
```

**步骤3：重启 Typora**

保存 CSS 文件后，重启 Typora 即可生效。

### 详细配置文件

完整的 CSS 配置（可选）：

```css
/* ==============================================
   ASCII 图形完美显示配置
   使用 Sarasa Mono SC 等宽字体
   ============================================== */

/* 代码块 */
.md-fences {
    font-family: 'Sarasa Mono SC', 'Consolas', 'Courier New', monospace !important;
    font-size: 13px;
    line-height: 1.6;
    background-color: #f6f8fa;
    border: 1px solid #e1e4e8;
    border-radius: 6px;
    padding: 16px;
}

/* 行内代码 */
code {
    font-family: 'Sarasa Mono SC', 'Consolas', 'Courier New', monospace !important;
    font-size: 0.9em;
    background-color: rgba(175, 184, 193, 0.2);
    padding: 0.2em 0.4em;
    border-radius: 3px;
}

/* 预格式化文本 */
pre {
    font-family: 'Sarasa Mono SC', 'Consolas', 'Courier New', monospace !important;
    font-size: 13px;
    line-height: 1.6;
}
```

---

## 🖥️ 其他编辑器配置

### VS Code

在 `settings.json` 中添加：

```json
{
  "editor.fontFamily": "'Sarasa Mono SC', Consolas, 'Courier New', monospace",
  "editor.fontSize": 14,
  "markdown.preview.fontFamily": "'Sarasa Mono SC', Consolas, 'Courier New', monospace"
}
```

### Obsidian

1. `设置` → `外观` → `字体`
2. 选择 `Sarasa Mono SC`

### Mark Text

1. `偏好设置` → `编辑器`
2. `代码块字体` → 选择 `Sarasa Mono SC`

---

## ✅ 验证显示效果

配置完成后，打开 `开发计划报告.md`，找到任意 ASCII 图形代码块，验证：

### 检查点

- [ ] 右边界完全垂直对齐（所有 `│` 和 `┐` 在同一列）
- [ ] 中文字符无重叠
- [ ] 中文字符无乱码
- [ ] ASCII 边框字符对齐
- [ ] 行与行之间对齐

### 示例验证

以下图形应该完美对齐：

```ascii
┌───────────────────────────────────┐
│        天气预报标题                │
├───────────────────────────────────┤
│                                   │
│     ┌─────────────────┐           │
│     │ 今日天气模块     │           │
│     │  - 日期         │           │
│     │  - 温度         │           │
│     │  - 描述         │           │
│     │  - 调节按钮      │           │
│     └─────────────────┘           │
│                                   │
└───────────────────────────────────┘
```

**如果右边界不对齐**：
1. 检查字体是否已安装
2. 检查 CSS 配置是否正确
3. 确认已重启编辑器

---

## 🎯 优势总结

### 开发体验
- ✅ **易于编辑**：直接在 Markdown 中修改
- ✅ **版本控制**：Git diff 可以清楚看到变化
- ✅ **快速迭代**：无需重新生成图片

### 文档质量
- ✅ **完美对齐**：使用等宽字体确保对齐
- ✅ **中英文混合**：Sarasa Mono SC 完美支持
- ✅ **文件体积**：纯文本，体积极小

### 维护成本
- ✅ **零依赖**：无需 Python、PIL 等工具
- ✅ **易维护**：修改文本即可
- ✅ **可移植**：只需配置字体

---

## 📦 备用方案

如果无法配置 Sarasa Mono SC 字体，仍可使用 PNG 图片方案：

### 生成图片
```bash
pip install Pillow
python generate_ascii_images.py
```

### 在文档中引用
```markdown
![手机端垂直布局](images/phone_layout.png)
```

所有图片已预生成在 `images/` 目录中，可直接使用。

---

## 🔗 相关资源

- **字体下载**: https://github.com/be5invis/Sarasa-Gothic/releases
- **配置说明**: `Typora字体配置说明.md`
- **图片生成脚本**: `generate_ascii_images.py`
- **开发计划报告**: `开发计划报告.md`

---

## ❓ 常见问题

### Q1: 为什么不直接用图片？
**A**: ASCII 代码块更易维护、体积小、版本控制友好，且使用等宽字体后显示效果同样完美。

### Q2: 如果没有安装 Sarasa Mono SC 怎么办？
**A**: 可以使用备用方案：PNG 图片（已预生成在 `images/` 目录）

### Q3: 其他 Markdown 编辑器如何配置？
**A**: 参考上文"其他编辑器配置"部分，或参考编辑器的字体设置文档。

### Q4: 配置后仍不对齐？
**A**: 
1. 确认 Sarasa Mono SC 已安装（双击 .ttf 文件安装）
2. 确认 CSS 已保存
3. 完全重启编辑器（不是刷新）
4. 检查 CSS 语法是否正确

---

**结论**: ✅ ASCII 代码块 + Sarasa Mono SC 是最优方案，兼顾显示效果和维护便利性！

