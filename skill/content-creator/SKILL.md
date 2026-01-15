---
name: content-creator
description: 智能内容创作工作流工具。从热榜抓取热点话题，使用 AI 生成多风格内容（Stanley/Defou/Combo），并进行爆款验证。当用户需要创作灵感、生成爆款内容或优化文章时使用。
---

# Content Creator - 智能内容创作助手

基于 Defou 方法论的智能内容创作工作流，帮助创作者从灵感获取到爆款验证的全流程自动化。

## 功能模块

### 1. 热点挖掘 (tophub)
从 TopHub 抓取全网实时热榜，AI 分析流量潜力话题。

```bash
./scripts/content-creator tophub [flags]
```

**输出**：
- 热榜数据 JSON
- AI 分析报告（选题建议、流量潜力评分）

### 2. 内容生成 (generate)
基于热点或用户草稿，生成三种风格的内容：
- **Version A (Stanley Style)**: 极致爆款，情绪共鸣，追求点击率
- **Version B (Defou Style)**: 深度认知，底层逻辑，长期价值
- **Version C (Combo Style)**: 融合版，传播节奏 + 深度内核

```bash
./scripts/content-creator generate [flags]
```

### 3. 爆款验证 (verify)
对生成的内容进行 6 大要素评估并自动优化：
- 好奇心 (Curiosity)
- 情绪共鸣 (Emotion)
- 价值感知 (Value)
- 时效性 (Timeliness)
- 内容节奏 (Rhythm)
- 新颖性 (Novelty)

```bash
./scripts/content-creator verify [flags]
```

### 4. 全自动模式 (auto)
一键完成：热点抓取 → 选题筛选 → 内容生成 → 爆款验证

```bash
./scripts/content-creator auto [flags]
```

## 全局参数

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--limit` | `-l` | 20 | 抓取热榜条目数量 |
| `--output-dir` | `-o` | `./outputs` | 输出目录 |
| `--format` | `-f` | `markdown` | 输出格式: markdown, json |
| `--api-key` | | `$ANTHROPIC_API_KEY` | Claude API Key |
| `--model` | | `claude-sonnet-4-5` | Claude 模型 |

## 使用示例

### 场景一：我没灵感，帮我找选题

```bash
# 抓取热榜并分析
./scripts/content-creator tophub -l 30 -o ./outputs

# 查看生成的分析报告
cat ./outputs/tophub_analysis_*.md
```

### 场景二：基于热点生成爆款内容

```bash
# 全自动生成（推荐）
./scripts/content-creator auto -l 20

# 或手动分步执行
./scripts/content-creator tophub -l 20
./scripts/content-creator generate --topic "从热榜选择的话题"
./scripts/content-creator verify --input ./outputs/generated_*.md
```

### 场景三：优化我的草稿

```bash
# 读取草稿并生成三版本
./scripts/content-creator generate --input ./my-draft.md

# 验证优化
./scripts/content-creator verify --input ./outputs/generated_*.md
```

### 场景四：批量处理文章链接

```bash
# 从链接列表文件批量处理
./scripts/content-creator batch --input ./links.txt
```

## 输出目录结构

```
outputs/
├── trends/                      # 热榜分析报告
│   └── tophub_analysis_*.md
├── generated/                   # 生成的内容
│   ├── stanley_*.md             # A版本（爆款风格）
│   ├── defou_*.md               # B版本（深度风格）
│   └── combo_*.md               # C版本（融合风格）
└── verified/                    # 验证优化后的内容
    └── verified_*.md
```

## 环境配置

需要设置 Claude API Key：

```bash
# 方式1：环境变量
export ANTHROPIC_API_KEY="sk-ant-..."

# 方式2：.env 文件
echo "ANTHROPIC_API_KEY=sk-ant-..." > .env

# 方式3：命令行参数
./scripts/content-creator auto --api-key "sk-ant-..."
```

## Prompt 模板参考

本 skill 包含以下参考文档（可自定义修改）：

- [references/stanley_style.md](references/stanley_style.md) - Stanley 风格 Prompt
- [references/defou_style.md](references/defou_style.md) - Defou 风格 Prompt
- [references/combo_style.md](references/combo_style.md) - Combo 风格 Prompt
- [references/viral_verification.md](references/viral_verification.md) - 爆款验证标准

## 技术栈

- **爬虫**: modu playwright 模块（支持 JS 渲染和去重）
- **AI**: Claude Sonnet 4.5 via Anthropic API
- **数据处理**: Go 原生 + encoding/json
- **输出格式**: Markdown, JSON

## 排重策略

使用 modu 的 playwright 模块内置去重功能：
- 基于 URL 的内容去重
- 基于标题相似度的去重
- 时间窗口内的重复检测

## 故障排查

**API Key 错误**:
```bash
export ANTHROPIC_API_KEY="your-key"
```

**热榜抓取失败**:
- 检查网络连接
- TopHub 可能需要等待几秒重试

**生成内容质量不佳**:
- 修改 `references/` 下的 Prompt 模板
- 调整 `--model` 参数使用更强大的模型

**Playwright 依赖未安装**:
```bash
# 安装 Playwright 浏览器
npx playwright install chromium
```

## 相关资源

- [Defou 方法论介绍](https://example.com/defou-methodology)
- [Stanley 风格写作指南](https://example.com/stanley-guide)
- [TopHub 热榜网站](https://tophub.today)
