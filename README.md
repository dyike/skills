# Skills

一个用于构建 AI Skills (技能) 的 Go 项目框架。Skills 是可被 Claude 等 AI 助手调用的独立工具模块，通过 SKILL.md 定义接口规范，包含可执行脚本和参考文档。

## 特性

- 🛠️ **模块化设计** - 每个 Skill 独立打包，包含 SKILL.md、脚本、参考文档和静态资源
- 🔧 **简单的构建系统** - 使用 Makefile 快速编译和安装
- 🎯 **多种内置 Skills** - content-creator、generate-image、news-scraper、sector、trade

## 项目结构

```
skills/
├── cmd/                   # 可执行程序入口
│   └── <skill-name>/      # 各 Skill 的 main 入口
├── skill/                 # Skill 定义
│   └── <skill-name>/
│       └── SKILL.md       # Skill 接口说明
├── internal/              # 内部包
├── models/                # 数据模型
├── Makefile               # 构建脚本
└── README.md
```

## 快速开始

### 查看帮助

```bash
make help
```

### 构建

```bash
# 构建指定 Skill
make build SKILL=<skill-name>

# 构建所有 Skills
make all
# 或
make build-all

# 查看可用 Skills
make list
```

### 安装

```bash
# 安装到 Gemini (~/.gemini/antigravity/skills)
make install SKILL=<skill-name> TARGET=gemini
# 或使用快捷方式
make gemini SKILL=<skill-name>

# 安装到 Claude (~/.claude/skills)
make install SKILL=<skill-name> TARGET=claude
# 或使用快捷方式
make claude SKILL=<skill-name>

# 安装到自定义路径
make install SKILL=<skill-name> TARGET=/path/to/skills

# 安装所有 Skills 到指定目标
make install-all TARGET=gemini
```

### 清理

```bash
# 清理构建目录
make clean
```

### 构建产物结构

```
build/<skill-name>/
├── SKILL.md              # AI 助手读取的接口说明
├── scripts/              # 可执行脚本/二进制
│   └── <skill-name>
├── references/           # 参考文档 (可选)
└── assets/               # 静态资源 (可选)
```

## 内置 Skills

| Skill | 说明 |
|-------|------|
| `content-creator` | 智能内容创作工具 |
| `generate-image` | AI 图像生成工具 |
| `news-scraper` | 多源新闻抓取工具 |
| `sector` | A股板块分析工具 |
| `trade` | 交易分析工具 |

> 各 Skill 的详细使用说明请查看对应的 `skill/<skill-name>/SKILL.md` 文件。

## 创建新 Skill

1. 在 `cmd/<skill-name>/` 创建 Go 程序入口
2. 在 `skill/<skill-name>/SKILL.md` 编写接口说明
3. (可选) 添加 `references/` 和 `assets/` 目录
4. 运行 `make build SKILL=<skill-name>` 构建

**SKILL.md 模板：**

```markdown
---
name: my-skill
description: 简短描述这个 Skill 的用途和使用场景
---

# My Skill

详细说明...

## Usage

\`\`\`bash
my-skill [options]
\`\`\`

## Examples

...
```

## 依赖

- Go 1.24+
- [Playwright](https://playwright.dev/) (可选，用于浏览器模式)

```bash
# 安装 Playwright 浏览器
npx playwright install chromium
```

## 许可证

MIT
