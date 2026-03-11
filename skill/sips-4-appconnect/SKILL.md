---
name: sips-4-appconnect
description: "Use this skill when the user needs to resize or convert screenshots for App Store Connect (Apple), or when they mention that real device screenshots don't meet App Store size requirements. Also trigger when the user asks about the macOS sips command, image resizing on Mac via terminal, or converting images to specific pixel dimensions. Key trigger: any mention of App Store Connect screenshot size mismatch, sips command, or Mac image resize terminal."
---

# sips — App Store Connect 截图尺寸转换

macOS 自带 `sips`（Scriptable Image Processing System），无需安装任何工具即可在终端直接调整图片尺寸。

## ⚠️ 关键注意事项：参数顺序是「先高后宽」

```bash
sips -z 高度 宽度 图片.png
```

`-z` 参数顺序是 **height width**，和直觉（宽×高）相反，非常容易写反！

---

## App Store Connect 常用尺寸

| 设备 | 尺寸（宽×高）| sips 命令参数（高 宽）|
|------|------------|----------------------|
| iPhone 6.9" (必须) | 1320×2868 | `-z 2868 1320` |
| iPhone 6.7" | 1290×2796 | `-z 2796 1290` |
| iPhone 6.5" | 1242×2688 | `-z 2688 1242` |
| iPhone 5.5" | 1242×2208 | `-z 2208 1242` |
| iPad Pro 13" | 2048×2732 | `-z 2732 2048` |
| iPad Pro 12.9" | 2048×2732 | `-z 2732 2048` |
| Mac | 2880×1800 | `-z 1800 2880` |
| Mac | 2560×1600 | `-z 1600 2560` |
| Mac | 1440×900   | `-z 900 1440` |
| Mac | 1280×800   | `-z 800 1280` |

---

## 常用命令示例

### 转换单张图片（覆盖原文件）
```bash
sips -z 2868 1320 screenshot.png
```

### 转换并输出到新文件（保留原图）
```bash
sips -z 2868 1320 screenshot.png --out screenshot_{pixelWidth}x{pixelHeight}.png
```

### 批量转换当前目录所有 PNG
```bash
for f in *.png; do
  sips -z 2868 1320 "$f" --out "converted_$f"
done
```

### 查看图片当前尺寸
```bash
sips -g pixelWidth -g pixelHeight screenshot.png
```

---

## 其他常用 sips 操作

```bash
# 转换格式（png → jpg）
sips -s format jpeg screenshot.png --out screenshot.jpg

# 旋转图片
sips -r 90 screenshot.png

# 按比例缩放（最长边不超过 1000px）
sips -Z 1000 screenshot.png
```

---

## 提醒

- `sips -z` 会改变宽高比（强制拉伸），如果原图比例不对，建议先裁剪再 resize
- App Store Connect 对截图有严格的像素要求，提交前用 `sips -g` 确认尺寸
- 真机截图与模拟器截图的 DPI/分辨率可能不同，以像素尺寸为准

