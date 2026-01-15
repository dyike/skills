# Content Creator Skill - 使用指南

## 简介

Content Creator 是一个基于 Defou 方法论的智能内容创作工作流工具，使用 Golang 编写，集成了 Claude API。

## 快速开始

### 1. 配置环境

设置 Claude API Key：

```bash
export ANTHROPIC_API_KEY="your-api-key-here"

# 或创建 .env 文件
echo "ANTHROPIC_API_KEY=your-api-key" > .env
```

### 2. 基本使用

#### 场景一：热点挖掘

从 TopHub 抓取热榜并使用 AI 分析流量潜力：

```bash
./scripts/content-creator tophub -l 30

# 输出示例：
# 🔍 正在抓取 TopHub 热榜 (limit: 30)...
# ✅ 成功抓取 30 条热点
# 🔄 去重后剩余 28 条
# 🤖 正在使用 AI 分析流量潜力...
# ✅ 分析完成
# 💾 分析报告已保存: ./outputs/trends/tophub_analysis_20260115_200630.md
```

生成的报告包括：
- 趋势总结
- 推荐选题（Top 5）
- 完整热点列表

#### 场景二：内容生成

基于话题生成三种风格的内容：

```bash
# 使用 Combo 风格（推荐）
./scripts/content-creator generate -t "AI 大模型降价潮" -s combo

# 指定输入文件
./scripts/content-creator generate -i my-draft.md -s combo

# 输出示例：
# 🎨 正在生成内容 (风格: combo)...
# ✅ 生成完成
# 💾 生成内容已保存: ./outputs/generated/combo_20260115_200730.md
```

可用的风格：
- `stanley`: 极致爆款风格（高传播度）
- `defou`: 深度认知风格（长期价值）
- `combo`: 融合风格（传播 + 深度）
- `all`: 生成所有三种风格

#### 场景三：爆款验证

对生成的内容进行 6 大要素评估：

```bash
./scripts/content-creator verify -i ./outputs/generated/combo_20260115_200730.md

# 输出示例：
# 🩺 正在验证内容...
# ✅ 验证完成
# 💾 验证报告已保存: ./outputs/verified/verified_20260115_200830.md
```

验证报告包括：
- 6 大要素评分（好奇心、情绪、价值、时效、节奏、新颖性）
- 详细诊断和问题分析
- 优化建议
- 自动优化重写版本

#### 场景四：全自动模式（推荐）

一键完成全流程：

```bash
./scripts/content-creator auto -l 20

# 流程：
# 🚀 启动全自动内容创作流程...
#
# 【步骤 1/3】抓取热点
# ✅ 抓取成功: 20 条热点
#
# 【步骤 2/3】生成内容
# 📝 选定话题: [AI 大模型降价潮]
# ✅ 生成完成
#
# 【步骤 3/3】验证优化
# ✅ 验证完成
#
# 🎉 全流程完成！生成文件: ./outputs/generated/combo_20260115_200930.md
```

### 3. 高级用法

#### 自定义 Prompt 模板

编辑 `references/` 目录下的 Prompt 文件：

```bash
# 修改 Combo 风格 Prompt
nano ~/.claude/skills/content-creator/references/combo_style.md

# 修改验证标准
nano ~/.claude/skills/content-creator/references/viral_verification.md
```

#### 使用浏览器模式（处理 JS 渲染）

```bash
# 注意：需要安装 Playwright
./scripts/content-creator tophub --use-browser
```

#### 指定输出目录

```bash
./scripts/content-creator auto -o ~/my-content-outputs
```

#### 使用不同的 Claude 模型

```bash
./scripts/content-creator auto --model claude-opus-4-5
```

## 输出目录结构

```
outputs/
├── trends/                          # 热榜分析
│   ├── tophub_analysis_*.md         # Markdown 报告
│   └── tophub_data_*.json           # 原始 JSON 数据
├── generated/                       # 生成的内容
│   ├── stanley_*.md                 # Stanley 风格
│   ├── defou_*.md                   # Defou 风格
│   └── combo_*.md                   # Combo 风格
└── verified/                        # 验证优化后的内容
    └── verified_*.md                # 验证报告 + 优化版本
```

## 在 Claude Code 中使用

安装后，Claude 会自动识别这个 skill。你可以直接对 Claude 说：

- "帮我找一些内容创作的选题"
- "基于这个话题生成一篇爆款文章"
- "验证一下这篇文章的爆款潜力"
- "我需要创作灵感"

Claude 会自动调用 content-creator skill 来完成任务。

## 技术栈

- **语言**: Go 1.24+
- **爬虫**: net/http + goquery (可扩展 playwright)
- **AI**: Claude Sonnet 4.5 via Anthropic API
- **CLI**: cobra 命令行框架
- **排重**: 基于标题的去重算法

## 故障排查

### API Key 错误

```bash
export ANTHROPIC_API_KEY="your-key"
```

### 抓取失败

- 检查网络连接
- TopHub 可能更新了 HTML 结构，需要更新选择器
- 尝试使用 `--use-browser` 模式

### 生成质量不佳

- 修改 `references/` 下的 Prompt 模板
- 尝试使用更强大的模型（如 opus）
- 调整 `--limit` 参数获取更多热点数据

## 开发和贡献

### 项目结构

```
skills/
├── cmd/content-creator/           # 主程序入口
│   └── main.go
├── internal/contentcreator/       # 核心逻辑
│   ├── cmd.go                     # Cobra 命令定义
│   ├── claude.go                  # Claude API 客户端
│   ├── tophub.go                  # 热点抓取
│   └── types.go                   # 数据类型
└── skill/content-creator/         # Skill 定义
    ├── SKILL.md                   # Claude 识别的接口
    └── references/                # Prompt 模板
        ├── combo_style.md
        └── viral_verification.md
```

### 构建

```bash
cd skills
make content-creator
```

### 安装

```bash
make install-content-creator
```

### 扩展功能

1. **添加新的数据源**: 在 `tophub.go` 中添加新的抓取函数
2. **新增风格**: 在 `references/` 添加新的 Prompt 模板
3. **改进验证标准**: 修改 `viral_verification.md`

## 许可证

MIT

## 相关资源

- [Defou 方法论](https://defou.com)
- [Claude API 文档](https://docs.anthropic.com)
- [TopHub 热榜](https://tophub.today)
