---
name: sector
description: A股板块分析工具。查看国内A股行业板块和概念板块的实时行情，获取热门板块推荐。
---

# A股板块分析

查看国内A股行业板块和概念板块的实时行情，获取热门板块推荐。

## Usage

```bash
# 查看板块列表
/sector list --type industry               # 行业板块
/sector list --type concept --limit 10     # 概念板块

# 查看热门板块
/sector hot --limit 5                      # 涨跌幅前5的板块

# 查看板块内个股
/sector stocks --name 光伏设备 --limit 15   # 光伏设备板块个股
```

## Commands

### list - 获取板块列表

获取行业板块或概念板块的实时行情列表。

```bash
sector list [--type industry|concept] [--limit 20]
```

**参数:**
| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--type, -t` | `industry` | 板块类型: `industry`(行业), `concept`(概念) |
| `--limit, -l` | `20` | 返回数量限制 |

**输出示例:**
```json
{
  "type": "industry",
  "count": 5,
  "sectors": [
    {
      "code": "BK0420",
      "name": "电力设备",
      "price": 1234.56,
      "change": 12.34,
      "change_rate": 2.15,
      "volume": 123456789,
      "amount": 45.67,
      "leader_stock": "宁德时代",
      "leader_rate": 5.23,
      "rise_count": 45,
      "fall_count": 12,
      "timestamp": "2026-01-25 10:30:00"
    }
  ],
  "timestamp": "2026-01-25 10:30:00",
  "summary": "行业板块共5个，上涨3个，下跌2个，平盘0个。涨幅最大: 电力设备(2.15%)"
}
```

---

### hot - 获取热门板块

获取涨跌幅前列的热门板块，包含涨幅和跌幅两个列表。

```bash
sector hot [--limit 10]
```

**参数:**
| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--limit, -l` | `10` | 每个列表返回数量限制 |

**输出示例:**
```json
{
  "top_rising": [
    {
      "name": "电力设备",
      "change_rate": 3.25,
      "leader_stock": "宁德时代"
    }
  ],
  "top_falling": [
    {
      "name": "房地产",
      "change_rate": -2.15,
      "leader_stock": "万科A"
    }
  ],
  "timestamp": "2026-01-25 10:30:00",
  "summary": "涨幅前三: 电力设备(+3.25%)、新能源汽车(+2.88%)、光伏(+2.45%)；跌幅前三: 房地产(-2.15%)、银行(-1.23%)、保险(-0.98%)"
}
```

---

### stocks - 获取板块内个股

获取指定板块内的个股列表，按涨跌幅排序。

```bash
sector stocks --name <板块名称> [--limit 20]
```

**参数:**
| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--name, -n` | 必填 | 板块名称，如: 光伏设备、新能源汽车 |
| `--limit, -l` | `20` | 返回数量限制 |

**输出示例:**
```json
{
  "sector_name": "光伏设备",
  "sector_code": "BK1031",
  "count": 10,
  "stocks": [
    {
      "code": "688223",
      "name": "晶科能源",
      "price": 6.90,
      "change": 1.15,
      "change_rate": 20.00,
      "volume": 54608700,
      "amount": 37.68,
      "turnover": 5.23,
      "pe": 12.5
    }
  ],
  "summary": "光伏设备板块共10只个股，上涨9只，下跌1只，涨停3只。领涨: 晶科能源(20.00%)"
}
```

## Data Source

数据来源: 东方财富行情中心
- 行业板块: https://quote.eastmoney.com/center/boardlist.html#industry_board
- 概念板块: https://quote.eastmoney.com/center/boardlist.html#concept_board

## Notes

- 数据为实时行情，交易时段内会持续更新
- 使用 Playwright 抓取，首次运行可能需要较长时间
- 非交易时段数据为上一交易日收盘数据
