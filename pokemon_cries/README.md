# 关都宝可梦叫声（Kanto Pokémon Cries）

本目录收录了关都图鉴（按关都图鉴编号）全部 153 只宝可梦的叫声音频。

## 内容

- `001.opus` ~ `153.opus`：以**关都图鉴编号**命名的叫声音频文件（Ogg Opus 格式）。
- `manifest.csv`：编号 ↔ 宝可梦名称 ↔ 原始叫声文件名 的对照表。

## 来源

音频取自神奇宝贝百科（52Poke Wiki）每只宝可梦详情页「叫声」处所播放的音频，
即各页面 `File:NNNN_cry.opus` 对应的原始 Opus 文件。

## 说明

- 编号 001–151 与全国图鉴编号一致；152、153 分别为美录坦、美录梅塔
  （其原始叫声文件为 `0808_cry.opus`、`0809_cry.opus`，见 `manifest.csv`）。
- 抓取脚本见 `../scripts/scrape_cries.py`。
