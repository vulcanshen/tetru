# Changelog

Each release names the version of terminu design (tdp) it follows.

## Unreleased

- Ten rows to a level again, and yours to set: every 5, 10, 15, 20 or 25 rows, on the Settings screen under the hint.
  A new setting is for the next game; a game going, or kept going, keeps the one it began with. A game kept by v0.2.0
  or before goes on at twenty.

## v0.2.0 — tdp v1.2.10

- `Esc` in a game only pauses it: no box over the paused game any more, `Enter` goes on, `Space` lists Resume, New
  game and End game.
- New game and End game run from the `Space` menu close it once done; it stayed open over the new game.
- Ending a game that would not be kept asks first, as restarting does.
- A game kept going as tetru quits keeps each block's colour, and comes back as it was: no more white blocks on the
  ground. A game kept by v0.1.0 still comes back, its blocks white.
- The hint, where a drop would set the piece, is a switch in Settings, under the keys: on at first, and on, it shows at
  every level, not the first five alone.
- Settings is one panel, the keys and the hint in two groups, KEYS and PLAY.
- Restarting a game that would not be kept (no points, or not in the best twenty) asks first: `r` pressed by mistake
  no longer throws the game away. `Esc` goes back to it.
- Twenty rows to a level, and fifteen levels: the speeds of the ten levels, a row a second to ten, are spread over the
  fifteen, so each level is a smaller step faster than the one before.
- The level stands big at both sides of the field, from 66 columns wide, with four rows clear above and below it and as
  wide as the sides allow: 1 to 15, or A to F for 10 to 15 under 78 columns, where two figures have no room. It lights
  up from the foot as you clear the level's twenty rows, and over what is lit waves go up, one after another, in the
  colour half way to the ground. When the level goes up, a band in the colour of the piece that cleared the last rows
  sweeps up over both sides with a fading trail and brings in the next one while you play on. The last level is all
  lit, in gold.

## v0.1.0 — tdp v1.2.8

The first stable version: a falling-block game for the terminal, for macOS, Linux and Windows, from Homebrew, Scoop
and the install scripts.

- A well twelve blocks wide and twenty-four deep, drawn as locku's tetromino saver draws its blocks, with the next two
  pieces, INFO (the level, the rows, the score) and KEYS beside it.
- Ten levels of speed, a level every ten rows. Up to level five, a lighter grey shows where the piece lands if you drop
  it now.
- One, two, three or four rows cleared at once each go their own way: fading, bursting, crumbling, or breaking to dust
  and going out to both sides; the stack above then comes down whole.
- The best twenty scores kept in `~/.tetru/scores.yaml` and ranked on the Rank screen, the first three in yellow, green
  and sky blue. Who played is asked as a game ends, ANON when nobody says; the game stays on screen till then.
- Every key you play with can be changed on the Settings screen, `Enter` and the moves too. The arrows keep what they
  do at first: move left to `a` and `←` still moves left. `Esc` pauses; `Space` shows what you can do.
- While playing, the footer shows the turns and the drop from 60 columns up, the turns as one pair, `d|k:rotate`; `?`
  lists every key, the turns apart.
- Quitting with a game going keeps it, nobody asked; the next start goes on with it, paused. Restart if you don't
  want it.

## v0.1.0-beta.2 — tdp v1.2.8

The second beta: more of the keys are yours to set, and the end of a game stays part of it.

- Settings: `Enter` and the moves can go to other keys too, and a letter so freed to another action. The arrows keep
  what they do at first: move left to `a` and `←` still moves left, and `?` writes it `a|←`.
- While playing, the footer writes the two turns as one pair, `d|k:rotate`, so the turns and the drop fit 60 columns
  whatever keys they are on; `q:quit` is in it too. `?` lists the turns apart, left and right.
- A game over stays in the game till its score is kept: `Esc` on the box that asks who played closes only the box, a
  second `Esc` leaves. The keys to play show greyed while there is nothing to move.
- `Tab` and the digits do nothing while playing, with no message: there is only one panel to go to.
- The Settings warning and log write keys in brackets: `Can't use [k] for Rotate left`.
- As a beta it is a GitHub pre-release only: download it from the release. Homebrew, Scoop and the install scripts
  get the first stable one.

## v0.1.0-beta.1 — tdp v1.2.4

The first version, a beta: a falling-block game for the terminal, for macOS, Linux and Windows.

- A well twelve blocks wide and twenty-four deep, drawn as locku's tetromino saver draws its blocks, with the next two
  pieces, INFO (the level, the rows, the score) and KEYS beside it.
- Ten levels of speed, a level every ten rows. Up to level five, a lighter grey shows where the piece lands if you drop
  it now.
- One, two, three or four rows cleared at once each go their own way: fading, bursting, crumbling, or breaking to dust
  and going out to both sides; the stack above then comes down whole.
- The best twenty scores kept in `~/.tetru/scores.yaml` and ranked on the Rank screen, the first three in yellow, green
  and sky blue. Who played is asked as a game ends, ANON when nobody says.
- Every key you play with can be changed on the Settings screen. `Esc` pauses; `Space` shows what you can do.
- Quitting with a game going keeps it, nobody asked; the next start goes on with it, paused. Restart if you don't
  want it.
- As a beta it is a GitHub pre-release only: download it from the release. Homebrew, Scoop and the install scripts,
  which take the latest stable release, get the first stable one.
