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

[Dungeon](https://dungeon.mamonas.dev) is a short turn-based roguelike that runs in your browser. Three floors, permadeath, ASCII art. You pick a class, explore the floor, try to get gear without dying, and move on to the next one. A run takes less than five minutes.

{{< image src="dungeon-gameplay.png" alt="Dungeon gameplay screenshot" width="960" renderWidth="600" class="center" >}}

---

A few years ago I took part in a game jam solo and built [Muscle Domain](https://keelaric.itch.io/muscle-domain) in Godot in a week. You play as a disembodied arm wielding a dumbbell. Throw it with left click, teleport to it with right click, collect steroids to reach the next level. I'm not sure it even runs anymore.

After that I wanted to make something again, but without a full engine and without worrying about art assets. Something small I could actually finish.

---

The two main inspirations are Rogue and Slay the Spire.

{{< image src="rogue-screenshot.png" alt="Rogue (1980) gameplay showing ASCII dungeon" width="637" renderWidth="550" class="center" >}}

[Rogue](https://en.wikipedia.org/wiki/Rogue_(video_game)) is the 1980 Unix game the whole genre is named after. Procedurally generated floors, gear that defines your build, permadeath, a dungeon drawn entirely in ASCII characters. Those mechanics are the backbone of Dungeon, and the aesthetic is a direct nod to it.

{{< image src="slay-the-spire.jpg" alt="Slay the Spire gameplay screenshot" width="1920" renderWidth="550" class="center" >}}

[Slay the Spire](https://www.megacrit.com/) gave me the event system. Random encounters mid-floor that offer you a choice with some risk attached: take the cursed item, pay gold to remove a debuff, gamble on a random reward. That kind of small decision-making does a lot to make short runs feel different from each other. Beyond those two, the list of roguelikes that had some influence is long: Hades, Dead Cells, Isaac, Risk of Rain. The genre has a lot of shared DNA and I wasn't trying to reinvent it.

---

**Why ASCII?** I'm genuinely bad at art. Text-based graphics remove that obstacle entirely. No sprites to draw, no animations, no art direction to maintain. Just characters on a grid. The constraint also pushed the design toward communicating everything through numbers and symbols, which is where the genre started anyway.

---

**Why Go + WASM?** The game doesn't need a real engine. It's a canvas, a render loop, and a turn system. I wanted to try compiling Go directly to the browser and this felt like the right project for it. `GOOS=js GOARCH=wasm`, and Go's `syscall/js` handles the bridge to the DOM. The render loop is `requestAnimationFrame` called from Go, audio goes through the Web Audio API, run history is stored in `localStorage`. No external libraries, about 6,600 lines of standard library Go.

One thing that surprised me was how clean the architecture stayed. A game is input, state update, render. WASM doesn't change any of that. The JS side is basically a thin shell that passes key events in, and Go handles everything else.

I initially wanted it to be playable on mobile too, with on-screen buttons and touch controls. That turned out to complicate the game design more than it was worth, so I cut it.

---

The project took about a week of evenings. I dropped it partway through when the balance felt off and I couldn't tell if it was fun or just frustrating. Came back later, played a few runs, decided it was actually fun, and pushed it through to something shippable.

It's not well balanced. I was the only playtester, which means I tuned everything around how I play. Some class and gear combinations are probably too strong. Some floors will end a run on turn three for no good reason.

---

What I'm happiest with is the complexity that comes out of fairly simple systems. Eight classes (four unlocked by winning three times with their base class), around forty gear items, a handful of synergies between specific item combinations. Each addition multiplies the number of viable builds in ways that are hard to predict when you're adding individual pieces. One synergy makes fire spread on every second strike. Another scales lifesteal off berserk stacks. I didn't plan for all the combinations, they just emerged. Watching a run come together around something unplanned is exactly why the genre works, and it's satisfying to see it happen even at this scale.

---

I've been playing video games since I was a kid and there's something different about being on the other side of it. Making even a small game gives you a better sense of why certain mechanics work the way they do, what tradeoffs the developers were making, and where the limitations come from. I'll probably keep making small games for that reason alone. The engine situation and the art situation remain unsolved problems, but those are for future me to deal with.

[Play it here.](https://dungeon.mamonas.dev) Runs in the browser, no install needed.

Thank you for reading.
