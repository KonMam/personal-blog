+++
title = "I made a roguelike"
date = "2026-03-21T13:55:10+03:00"
tags = ["gamedev", "go", "webassembly"]
categories = ["tech"]
draft = true
description = """
I built a small ASCII roguelike in Go and WebAssembly. Here is why, and what I learned making it.
"""
+++

[Dungeon](https://dungeon.mamonas.dev) is a short turn-based roguelike that runs in your browser. Three floors, permadeath, ASCII art. A run takes less than five minutes.

---

This isn't my first game. A few years ago I took part in a game jam solo and built [Muscle Domain](https://keelaric.itch.io/muscle-domain) in Godot in a week. You play as a disembodied arm wielding a dumbbell. Throw it with left click, teleport to it with right click, collect steroids to reach the next level. I'm not sure it even runs anymore.

After that I wanted to make something again, but without a full engine and without worrying about art assets. Something small I could actually finish.

---

The two main inspirations are Rogue and Slay the Spire.

{{< image src="rogue-screenshot.png" alt="Rogue (1980) gameplay showing ASCII dungeon" width="637" renderWidth="800" >}}

[Rogue](https://en.wikipedia.org/wiki/Rogue_(video_game)) is the 1980 Unix game the whole genre is named after. Procedurally generated floors, gear that defines your build, permadeath. Those mechanics are the backbone of Dungeon.

{{< image src="slay-the-spire.jpg" alt="Slay the Spire gameplay screenshot" width="1920" renderWidth="800" >}}

[Slay the Spire](https://www.megacrit.com/) gave me the event system. Random encounters with a choice attached, usually involving some risk. That kind of small decision making does a lot of work in a short run. Beyond those two, the list of roguelikes that had some influence is long: Hades, Dead Cells, Isaac. The genre has a lot of shared DNA and I wasn't trying to reinvent it.

---

**Why ASCII?** I'm genuinely bad at art. Text-based graphics remove that obstacle entirely. No sprites, no animations, no art direction needed. Just characters on a grid, which as it turns out is fine.

---

**Why Go + WASM?** The game doesn't need a real engine. It's a canvas, a render loop, and a turn system. I wanted to try compiling Go directly to the browser and this felt like the right project for it. `GOOS=js GOARCH=wasm`, and Go's `syscall/js` handles the bridge to the DOM. No external libraries, about 6,600 lines of standard library Go.

I initially wanted it to be playable on mobile too, with on-screen buttons and touch controls. That complicated things more than it was worth and started limiting what I could do with the game design, so I cut it.

---

The project took about a week of evenings. I dropped it partway through when the balance felt broken and I couldn't tell if it was fun or just frustrating. Came back later, played a few runs, decided it was fun, and pushed it to something shippable.

It's not well balanced. I was the only playtester. Some class and gear combinations are probably too strong, some floors will end a run on turn three for no real reason. That's fine. It's a side project.

---

What I'm happiest with is the complexity that emerges from fairly simple systems. Eight classes, around forty gear items, a handful of synergies between specific item combinations. Each addition multiplies the number of viable builds in ways that are hard to predict when you're adding individual pieces. Watching a run come together around an unplanned combination is exactly why the genre works, and it's satisfying to see that happen even at this scale.

{{< image src="dungeon-gameplay.png" alt="Dungeon gameplay screenshot" width="960" renderWidth="800" >}}

---

[Play it here.](https://dungeon.mamonas.dev) Runs in the browser, no install needed.

Thank you for reading.
