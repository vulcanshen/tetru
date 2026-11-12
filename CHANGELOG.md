# Changelog

Each release names the version of terminu design (tdp) it follows.

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
- Save keeps the game and quits; the next start goes on with it.
- As a beta it is a GitHub pre-release only: download it from the release. Homebrew, Scoop and the install scripts,
  which take the latest stable release, get the first stable one.
