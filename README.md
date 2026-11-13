# tetru

[![Release](https://img.shields.io/github/v/release/vulcanshen/tetru)](https://github.com/vulcanshen/tetru/releases)
[![License](https://img.shields.io/github/license/vulcanshen/tetru)](LICENSE)

**Language**: English · [繁體中文](README-zh_TW.md)

**A falling-block game for the terminal.**

Pieces of four blocks fall into a well twelve blocks wide and twenty-four deep; move and turn them as they fall, and
fill rows from wall to wall to clear them. Every twenty rows is a level, and every level falls faster, up to the fifteenth. tetru keeps the best twenty scores
with the player's name and the time, and ranks them, the best first.

![tetru: a game played, the Space menu pausing it, and the score kept](docs/demo.gif)

## Features

- **Drawn as locku's falling blocks**: the same square pixels and colours. The well at the left; at its right, in the same
  panel, the next two pieces, a box each, then INFO with the level, the rows and the score, and KEYS with the keys to
  play with.
- **Your keys**: every key you play with can be changed on the Settings screen; `Esc` (pause) and `Space` (the menu)
  stay as they are.
- **Fifteen levels of speed**: a level every twenty rows, from a row a second to ten a second; past the fifteenth the
  speed stays.
- **The level beside the field**: from 66 columns wide, the level stands big at both sides of the field, four rows
  clear of its top and its foot, and lights up from the foot as you clear its twenty rows, waves going up it; a
  band in the colour of the piece that cleared the last rows sweeps up both sides to bring in the next. Past 9 the
  levels are A to F, and the last, F, is gold.
- **A hint, if you like**: a lighter grey on the ground shows where the piece lands if you drop it now, at every
  level; switch it off on the Settings screen.
- **The best twenty kept**: the place, the name, the score and the time, the best first on the Rank screen, the first
  three in yellow, green and sky blue. You play first and say who played as the game ends; with no name it is ANON. A
  game whose score is not in the best twenty is not kept, and nobody is asked.
- **Pause whenever**: `Esc` pauses and leaves the game where it is, a box saying the level, the rows and the score,
  `Enter` to go on. `Space` shows what you can do, the game waiting while it is open.
- **Quit and go on later**: quitting with a game going keeps it, its score and the blocks on the ground, nobody
  asked; the next start goes on with it, paused, the blocks on the ground white. Don't want it? Restart.

## Install

### Requirements

- macOS, Linux (WSL too) or Windows. On Windows, Windows Terminal: the old console has no truecolor.
- A [Nerd Font](https://www.nerdfonts.com) terminal profile: the blocks and the frames are Nerd Font glyphs.
- A truecolor (24-bit) terminal: the faded colours and the layer colours cannot be told apart in 256 colours.
- A terminal at least 42 columns wide and 32 rows high. Under 60 columns the well shows alone; from 60 on, the pieces
  to come, INFO and KEYS show beside it. In a
  smaller one tetru shows only its name, the size it needs and how to quit, and comes back as it was once the terminal
  grows: the game waits meanwhile.

tetru measures how many cells an icon and a powerline symbol take when it starts. When it gets them wrong, set them:

| Variable | What it sets |
|---|---|
| `TETRU__ICON_WIDTH` | The cells an icon takes, for tetru |
| `TETRU__POWERLINE_WIDTH` | The cells a powerline symbol takes, for tetru |
| `TERMINU__ICON_WIDTH` | The cells an icon takes, for every terminu family app: set it once and they all read it |
| `TERMINU__POWERLINE_WIDTH` | The cells a powerline symbol takes, for every terminu family app |
| `LC_TERMINU_ICON_WIDTH` | The cells an icon takes, brought over ssh by a family app with ssh |
| `LC_TERMINU_POWERLINE_WIDTH` | The cells a powerline symbol takes, brought over ssh by a family app with ssh |

For an icon and for a powerline symbol each, tetru takes the first one set in the order of the table, and measures
only when none is. Run inside a family app's terminal, tetru gets the `TERMINU__` ones from the app around it; reached
over ssh from one, the `LC_TERMINU_` ones.

### Homebrew (macOS / Linux)

```sh
brew install vulcanshen/tap/tetru
```

### Scoop (Windows)

```powershell
scoop bucket add vulcanshen https://github.com/vulcanshen/scoop-bucket
scoop install tetru
```

### Script

macOS / Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/vulcanshen/tetru/main/install.sh | sh
```

No sudo: it installs to `~/.local/bin`.

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/vulcanshen/tetru/main/install.ps1 | iex
```

It installs to `%LOCALAPPDATA%\tetru` and adds it to your `PATH`.

### The first start

tetru opens on the Tetru screen with the focus on the well, ready to play. It writes nothing until a game with points
ends, then `~/.tetru/scores.yaml`, or until a key is changed, then `~/.config/tetru/config.yaml`, or until tetru quits
with a game going, then `~/.tetru/game.yaml`. On Windows `~` is `%USERPROFILE%`.

### Uninstall

```sh
brew uninstall tetru
```

```powershell
scoop uninstall tetru
```

Installed with the script:

```sh
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/vulcanshen/tetru/main/uninstall.sh | sh

# Windows PowerShell
irm https://raw.githubusercontent.com/vulcanshen/tetru/main/uninstall.ps1 | iex
```

It asks before removing `~/.tetru`, your scores.

## Quick start

1. Run `tetru`.
2. Press `Enter`: the game starts.
3. `h` and `l` move the piece, `d` and `k` turn it, `j` takes it down a row and `Enter` drops it.
4. Press `Space` to see what you can do, the game waiting; `Esc` closes the menu and the game goes on.
5. When the game is over, type your name and press `Enter`, or just `Enter` for ANON: the score is kept.
6. Press `q` to quit; a game going is kept, and the next start goes on with it.

## Usage

```
tetru            play
tetru version    print the version (-v, --version)
tetru help       print the command-line usage (-h, --help)
```

tetru has three screens; switch with their keys, or from Global operation in the `Space` menu:

| Key | Screen |
|---|---|
| `T` | Tetru: the game |
| `R` | Rank: the best twenty scores, each with its place |
| `S` | Settings: the keys to play with, and the hint |

Playing (the game in the `Gaming` mode), with the keys as they first are:

| Key | What it does |
|---|---|
| `h`, `←` | Move the piece left |
| `l`, `→` | Move it right |
| `j`, `↓` | Move it down a row: a point a row |
| `d`, `PgDn` | Turn it left (counterclockwise) |
| `k`, `↑` | Turn it right (clockwise) |
| `Enter` | Drop it to the bottom and set it there: two points a row |
| `r` | Restart: a new game; with points, after asking who played this one, its score kept |
| `Space` | Show what you can do (restart, quit), the game waiting; close it to go on |
| `Esc` | Pause, out of the game, where it stands in a box; `Enter` goes on |

On the Settings screen, `Enter` on an action lists the keys: the small letters and `Enter`, but for `q` and `r`,
which quit and restart. The one chosen is the key from then on, and KEYS beside the well shows it; another
action's key is refused with a warning saying whose it is. `Enter` and the moves can go to other keys too, and a
letter so freed to another action. The arrows are no one's to choose and keep what they do at first: set left to `a`
and `←` still moves left. `Esc` always pauses and `Space` always opens the menu. Under the keys, PLAY has the hint's
switch: `Enter` there switches it off or on, kept from then on.

Clearing 1, 2, 3 or 4 rows at once scores 100, 300, 500 or 800 times the level. A piece that lies on the stack is set half
a second later; moving or turning it gives it the half second again, a few times.

On the PLAYGROUND panel, out of the game:

| Key | What it does |
|---|---|
| `Enter` | Play a new game at once, or go on with the one paused |
| `n` | A new game; with one going, after asking who played it, its score kept |
| `x` | End the game going, after asking who played it, its score kept |

The keys every family app shares:

| Key | What it does |
|---|---|
| `Space` | What you can do here: on this item, on this panel, and for the whole app |
| `?` | Every key that works here |
| `!` | The App Log: what was done, newest last; here, the scores kept |
| `j\|k`, `gg\|G` | On Rank and Settings: move, to the top, to the bottom |
| `Esc` | Close the popup on top, one at a time; it never leaves tetru |
| `q` | Quit; a game going is kept, and the next start goes on with it |

## Settings and data

| What | Where | To move it |
|---|---|---|
| The scores | `~/.tetru/scores.yaml` | `TETRU__DATA` |
| A game kept going as tetru quit, till the next start takes it up | `~/.tetru/game.yaml` | `TETRU__DATA` |
| The keys to play with, and the hint | `~/.config/tetru/config.yaml` (`$XDG_CONFIG_HOME/tetru`) | `TETRU__CONFIG` |

The best twenty scores are kept; a game with no points, or too few for them, is not. A game kept going is kept in the rank once it ends, after it is taken up again. tetru uses none of the family's shared setting keys.

## Limits

- A score is kept only once its game ends: when it reaches the top, or is ended or a new one started. Quitting, or
  closing the terminal, keeps the game going for the next start; a game in a program killed outright (`kill -9`, a
  power cut) is lost.

## Links

- [CHANGELOG](CHANGELOG.md)
- [locku](https://github.com/vulcanshen/locku): the screensaver whose falling blocks tetru is drawn as

## The terminu family

tetru follows terminu design:

- [terminu design](https://github.com/vulcanshen/terminu): the family's design rules
- [terminu-go](https://github.com/vulcanshen/terminu-go): the Go library tetru is built with
- [terminu-harness](https://github.com/vulcanshen/terminu-harness): the Claude Code plugin for building apps to terminu
  design

## License

[GPL-3.0](LICENSE)
