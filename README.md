# Skills

一个用于构建 AI Skills (技能) 的 Go 项目框架。Skills 是可被 Claude 等 AI 助手调用的独立工具模块，通过 SKILL.md 定义接口规范，包含可执行脚本和参考文档。

## 特性

- 🛠️ **模块化设计** - 每个 Skill 独立打包，包含 SKILL.md、脚本、参考文档和静态资源
- 🔧 **简单的构建系统** - 使用 Makefile 快速编译和安装
- 📰 **内置 news-scraper** - 从 Hacker News、Product Hunt、TLDR 等来源抓取科技新闻

## 项目结构

```
skills/
├── cmd/                   # 可执行程序入口
│   └── news-scraper/      # 新闻抓取器
├── skill/                 # Skill 定义
│   └── news-scraper/
│       └── SKILL.md       # Skill 接口说明
├── internal/              # 内部包
│   └── scraper/           # 抓取器实现
├── models/                # 数据模型
├── Makefile               # 构建脚本
└── README.md
```

## 快速开始

### 构建

```bash
# 构建指定 Skill
make news-scraper

# 构建所有 Skills
make all

# 查看可用 Skills
make list
```

### 安装

```bash
# 安装到默认路径 (~/.claude/skills)
make install-news-scraper

# 安装到自定义路径
make install-news-scraper INSTALL_DIR=/path/to/skills
```

### 构建产物结构

```
~/.claude/skills/news-scraper/
├── SKILL.md              # AI 助手读取的接口说明
├── scripts/              # 可执行脚本/二进制
│   └── news-scraper
├── references/           # 参考文档 (可选)
└── assets/               # 静态资源 (可选)
```

## 内置 Skills

### news-scraper

从多个科技新闻来源抓取内容。

**支持的来源：**

| 来源 | 标志 | 说明 |
|------|------|------|
| Hacker News | `hn` | 头条新闻与评分 |
| Product Hunt | `ph` | 今日产品与投票 |
| Newsletter | `newsletter` | 自定义 Newsletter 归档 |
| Substack | `substack` | Substack 出版物 |
| TLDR | `tldr` | TLDR 技术简报 (tech/ai/webdev/crypto/devops/founders) |

**使用示例：**

```bash
# 抓取 Hacker News 前 20 条
news-scraper -source hn -limit 20

# 多来源抓取，输出 Markdown
news-scraper -source hn -source ph -format markdown

# 抓取 Substack
news-scraper -source substack -substack-name stratechery -limit 10

# 抓取 TLDR AI 简报
news-scraper -source tldr -tldr-category ai

# 使用 Playwright 浏览器模式 (处理 JS 渲染)
news-scraper -source ph -use-browser

# 输出到文件
news-scraper -source hn -format json -o news.json
```

**输出格式：**
- `text` - 纯文本编号列表 (默认)
- `markdown` - Markdown 带链接
- `json` - JSON 数组

## 创建新 Skill

1. 在 `cmd/<skill-name>/` 创建 Go 程序入口
2. 在 `skill/<skill-name>/SKILL.md` 编写接口说明
3. (可选) 添加 `references/` 和 `assets/` 目录
4. 运行 `make <skill-name>` 构建

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
