# tetru

[![Release](https://img.shields.io/github/v/release/vulcanshen/tetru)](https://github.com/vulcanshen/tetru/releases)
[![License](https://img.shields.io/github/license/vulcanshen/tetru)](LICENSE)

**Language**: [English](README.md) · 繁體中文

**在終端機裡玩的落塊消行遊戲。**

四格一組的方塊往一個十二格寬、二十四格深的井裡掉，趁它落下時左右移動、旋轉，把一整行從左牆排滿到右牆就會消掉。每消十行升一關，
一關比一關掉得快，到第十關為止。最高的二十個分數會連同玩家的名字與時間記下來，由高到低排名。

![tetru：玩一局、按 Space 暫停並打開 menu、分數記進排行](docs/demo.gif)

## 特色

- **畫法跟 locku 的落塊螢幕保護一樣**：一樣的方形像素與顏色。左邊是井；右邊在同一個 panel 裡，是接下來的兩個方塊，各一個框，
  再下面是 INFO（關卡、消了幾行、分數）與 KEYS（玩的時候用的鍵）。
- **按鍵自己定**：玩的時候用的鍵都能在 Settings 畫面改；只有 `Esc`（暫停）與 `Space`（menu）不能改。
- **十關速度**：每消十行升一關，從一秒落一行快到一秒落十行；過了第十關就不再加速。
- **場地兩邊的關卡大字**：終端機有 66 欄以上時，場地左右各立著一個跟場地一樣高的關卡數字，這一關每消一行，就從底下往上
  亮一成；升關時一大片白光由下往上掃過兩邊，換上下一關的數字。第十關是 10（不到 78 欄時是 A），會像心跳一樣一下一下地跳。
- **前五關有提示**：場地底上淺一點的灰色，畫出方塊現在按 `Enter` 直接落到底會停在哪裡；第六關起就沒有提示。
- **記下前二十名**：名次、名字、分數與時間，在 Rank 畫面由高到低排，前三名分別是黃、綠、天藍色的字。先玩，一局結束時才問是誰玩的；不填就是 ANON。
  分數擠不進前二十名的一局不記，也不問名字。
- **隨時暫停**：`Esc` 暫停並先離開遊戲，跳出一個框寫著關卡、行數與分數，這一局原封不動等你按 `Enter` 回來。
  `Space` 列出這裡能做什麼，開著的時候遊戲也會停。
- **離開了下次接著玩**：還有一局在玩時離開，tetru 會自己把這一局的分數與地上的方塊存起來，不問；下次打開 tetru 就接著玩（先停著），地上的方塊畫成白色。不想玩這一局就重新開始。

## 安裝

### 前置需求

- macOS、Linux（WSL 也可以）或 Windows。Windows 上請用 Windows Terminal：舊的主控台沒有 truecolor。
- 終端機用 [Nerd Font](https://www.nerdfonts.com)：方塊與框線都是 Nerd Font 的字。
- 終端機支援 truecolor（24-bit 色）：256 色分不出淡化與各層 popup 的顏色。
- 終端機至少 42 欄寬、32 列高。不到 60 欄時只畫井；60 欄以上，接下來的方塊、INFO 與 KEYS 會並排在旁邊。比這個小時，tetru 只畫出名字、需要的大小
  與怎麼離開，遊戲先停著；終端機拉大就回到原本的畫面。

tetru 啟動時會自己量一個 icon、一個 powerline 符號佔幾格。量錯時可以直接指定：

| 變數 | 指定什麼 |
|---|---|
| `TETRU__ICON_WIDTH` | icon 佔幾格，只給 tetru |
| `TETRU__POWERLINE_WIDTH` | powerline 符號佔幾格，只給 tetru |
| `TERMINU__ICON_WIDTH` | icon 佔幾格，terminu 家族每個 app 共用：設一次，每個 app 都讀得到 |
| `TERMINU__POWERLINE_WIDTH` | powerline 符號佔幾格，terminu 家族每個 app 共用 |
| `LC_TERMINU_ICON_WIDTH` | icon 佔幾格，由有 ssh 的家族 app 連線時帶過來 |
| `LC_TERMINU_POWERLINE_WIDTH` | powerline 符號佔幾格，由有 ssh 的家族 app 連線時帶過來 |

icon 與 powerline 符號各自照表上的順序，用第一個有設的；都沒設才自己量。在家族 app 的終端機裡跑 tetru 時，外層的 app
會替它設好 `TERMINU__` 的兩個；從家族 app 經 ssh 連過來時，拿到的是 `LC_TERMINU_` 的兩個。

### Homebrew（macOS / Linux）

```sh
brew install vulcanshen/tap/tetru
```

### Scoop（Windows）

```powershell
scoop bucket add vulcanshen https://github.com/vulcanshen/scoop-bucket
scoop install tetru
```

### 安裝腳本

macOS / Linux：

```sh
curl -fsSL https://raw.githubusercontent.com/vulcanshen/tetru/main/install.sh | sh
```

不用 sudo：裝在 `~/.local/bin`。

Windows PowerShell：

```powershell
irm https://raw.githubusercontent.com/vulcanshen/tetru/main/install.ps1 | iex
```

裝在 `%LOCALAPPDATA%\tetru`，並把它加進 `PATH`。

### 第一次啟動

一打開就是 Tetru 畫面，focus 在井上，可以直接開始玩。有分數的一局結束之前，tetru 什麼都不寫：第一次記分數時才建立
`~/.tetru/scores.yaml`，第一次改按鍵時才建立 `~/.config/tetru/config.yaml`，還有一局在玩時離開才建立 `~/.tetru/game.yaml`。Windows 上的 `~` 是 `%USERPROFILE%`。

### 移除

```sh
brew uninstall tetru
```

```powershell
scoop uninstall tetru
```

用安裝腳本裝的：

```sh
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/vulcanshen/tetru/main/uninstall.sh | sh

# Windows PowerShell
irm https://raw.githubusercontent.com/vulcanshen/tetru/main/uninstall.ps1 | iex
```

刪掉 `~/.tetru`（你的分數紀錄）之前會先問。

## 快速開始

1. 執行 `tetru`。
2. 按 `Enter`，遊戲開始。
3. `h`、`l` 左右移動，`d`、`k` 旋轉，`j` 往下一格，`Enter` 直接落到底。
4. 按 `Space` 看看能做什麼，遊戲先停著；`Esc` 關掉 menu，遊戲接著走。
5. 遊戲結束時打你的名字再按 `Enter`，或直接按 `Enter` 記成 ANON，分數就記下來了。
6. 按 `q` 離開；還有一局在玩時會存起來，下次打開接著玩。

## 使用方式

```
tetru            開始玩
tetru version    印出版本（-v、--version）
tetru help       印出命令列的用法（-h、--help）
```

tetru 有三個畫面，按它的鍵切換，或從 `Space` menu 的 Global operation 換：

| 鍵 | 畫面 |
|---|---|
| `T` | Tetru：遊戲 |
| `R` | Rank：前二十名的分數，每一筆有名次 |
| `S` | Settings：玩的時候用的鍵 |

玩的時候（遊戲在 `Gaming` 模式裡），一開始的鍵：

| 鍵 | 做什麼 |
|---|---|
| `h`、`←` | 方塊往左移一格 |
| `l`、`→` | 往右移一格 |
| `j`、`↓` | 往下一格：每一格 1 分 |
| `d`、`PgDn` | 往左轉（逆時針） |
| `k`、`↑` | 往右轉（順時針） |
| `Enter` | 直接落到底並固定：每一格 2 分 |
| `r` | 重新開始：開新的一局；有分數時先問是誰玩的，並記下這一局的分數 |
| `Space` | 列出能做什麼（重新開始、離開），遊戲先停著；關掉就接著玩 |
| `Esc` | 暫停並離開遊戲，跳出框寫著現在的狀況；按 `Enter` 繼續 |

在 Settings 畫面的一個動作上按 `Enter`，會列出可以選的鍵：小寫字母與 `Enter`，但不列 `q`、`r`（離開、重新開始）。
選了之後就換成那個鍵，井旁邊的 KEYS 也跟著寫；選到別的動作的鍵，會跳出警告說是誰的鍵，鍵不變。`Enter` 和移動也能
改到別的鍵，空出來的字母可以給別的動作。方向鍵不能選，一直做一開始的事：把左移改成 `a`，`←` 照樣往左。`Esc` 一直是暫停，
`Space` 一直是打開 menu。

一次消 1、2、3、4 行，分別得 100、300、500、800 分，再乘上當時的關卡。方塊碰到底之後過半秒才固定；這段時間裡移動或
旋轉，會再給半秒，但次數有限。

在 PLAYGROUND panel 上、沒在玩的時候：

| 鍵 | 做什麼 |
|---|---|
| `Enter` | 直接開新的一局；有暫停中的一局時，接著玩那一局 |
| `n` | 開新的一局；還有一局沒結束時先問是誰玩的，並記下那一局的分數 |
| `x` | 結束還在玩的這一局，先問是誰玩的，並記下分數 |

家族每個 app 都一樣的鍵：

| 鍵 | 做什麼 |
|---|---|
| `Space` | 這裡能做什麼：對這一項、對這個 panel、對整個 app |
| `?` | 這裡能按的每一個鍵 |
| `!` | App Log：做過的事，最新的在最下面；在 tetru 裡是記下的每一筆分數 |
| `j\|k`、`gg\|G` | 在 Rank 與 Settings 上移動、到最上面、到最下面 |
| `Esc` | 一次關掉最上層的一個 popup；不會離開 tetru |
| `q` | 離開；還有一局在玩時會存起來，下次打開接著玩 |

## 設定與資料

| 什麼 | 在哪裡 | 換位置 |
|---|---|---|
| 分數紀錄 | `~/.tetru/scores.yaml` | `TETRU__DATA` |
| 離開時還在玩的那一局，下次啟動接著玩之前 | `~/.tetru/game.yaml` | `TETRU__DATA` |
| 玩的時候用的鍵 | `~/.config/tetru/config.yaml`（`$XDG_CONFIG_HOME/tetru`） | `TETRU__CONFIG` |

只記前二十名的分數；0 分或擠不進前二十名的一局不記。離開時存起來的一局，下次接著玩完、結束時才記進排行。tetru 沒有用到家族共用的設定 key。

## 限制

- 分數要一局結束才會記下：堆到頂、結束這一局或開新局。離開 tetru 或關掉它所在的終端機時，還在玩的一局會存起來下次接著玩；
  程式被直接砍掉（`kill -9`、斷電）時，那一局會不見。

## 相關連結

- [CHANGELOG](CHANGELOG.md)
- [locku](https://github.com/vulcanshen/locku)：tetru 的畫法來自它的落塊螢幕保護

## terminu family

tetru 遵循 terminu design：

- [terminu design](https://github.com/vulcanshen/terminu)：家族的設計規範
- [terminu-go](https://github.com/vulcanshen/terminu-go)：tetru 用來做畫面的 Go 函式庫
- [terminu-harness](https://github.com/vulcanshen/terminu-harness)：照 terminu design 做 app 的 Claude Code plugin

## License

[GPL-3.0](LICENSE)
