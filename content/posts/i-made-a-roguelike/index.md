+++
title = "I made a tiny roguelike"
date = "2026-03-22T13:55:10+03:00"
tags = ["gamedev", "go", "webassembly"]
categories = ["gaming"]
draft = false
description = """
I built Dungeon, a short ASCII roguelike in Go and WebAssembly. Here is how it came together.
"""
+++

[Dungeon](https://dungeon.mamonas.dev) is a short turn-based roguelike that runs in your browser. Three floors, permadeath, ASCII art. You pick a class, explore the floor, try to get gear without dying, and move on to the next one. A run takes less than five minutes.

{{< image src="dungeon-gameplay.png" alt="Dungeon gameplay screenshot" width="960" renderWidth="600" class="center" >}}

---

A few years ago I took part in a game jam solo and built [Muscle Domain](https://keelaric.itch.io/muscle-domain) in Godot in a week. You play as a disembodied arm wielding a dumbbell. Throw it with left click, teleport to it with right click, collect "steroids" to reach the next level. It still works, though not on all browsers.

{{< image src="muscledomain-title.png" alt="Muscle Domain title screen" width="1876" renderWidth="550" class="center" >}}

After that I realised how much effort went into something so simple and didn't make anything new for a while. But I kept watching devlog channels on YouTube and reading other people's game dev posts, and eventually decided to give it another go. This time without a big engine, and without anything that would require me to be good at art.

---

The genre was an easy pick. Roguelikes have been eating my brain for years, and two in particular shaped what Dungeon ended up being.

{{< image src="rogue-screenshot.png" alt="Rogue (1980) gameplay showing ASCII dungeon" width="637" renderWidth="550" class="center" >}}

[Rogue](https://en.wikipedia.org/wiki/Rogue_(video_game)) is the 1980 Unix game the whole genre is named after. Procedurally generated floors, gear that defines your build, permadeath, a dungeon drawn entirely in ASCII characters. Those mechanics are the backbone of Dungeon, and the aesthetic is a direct nod to it.

{{< image src="slay-the-spire.jpg" alt="Slay the Spire gameplay screenshot" width="1920" renderWidth="550" class="center" >}}

[Slay the Spire](https://www.megacrit.com/) gave me the event system. Random encounters mid-floor that offer you a choice with some risk attached: take the cursed item, pay gold to remove a debuff, gamble on a random reward. That kind of small decision-making does a lot to make short runs feel different from each other.

Beyond those two, the list of roguelikes that had some influence is long: Hades, Dead Cells, Isaac, Risk of Rain. The genre has a lot of shared DNA and I wasn't trying to reinvent it.

---

With the game figured out, I still had to decide how to build it. I'm genuinely bad at art, so making it ASCII removed that obstacle entirely. No sprites to draw, no animations, no art direction. Just characters on a grid, which also happens to be exactly where the genre started.

For the tech I went with Go compiled to WebAssembly. WASM is a binary format that browsers can run at near-native speed -- code compiled from Go, C, Rust, and others runs directly in the browser without plugins or JavaScript. For something this simple I didn't need a real engine, so it felt like a good excuse to try it. The Go code talks to the browser through a small set of APIs:

<div class="diagram-scroll">
<figure class="center" style="width:100%;"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 580 170" width="100%" aria-label="Dungeon tech stack: renderer connects to Canvas 2D, sound to Web Audio API, run history to localStorage, all bridged via syscall/js">
  <defs><marker id="dg-arr" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="5" markerHeight="5" orient="auto-start-reverse"><path d="M 0 2 L 8 5 L 0 8 z" fill="currentColor" fill-opacity="0.35"/></marker></defs>
  <text x="100" y="15" text-anchor="middle" fill="#6C8CFF" font-size="11" font-weight="600">Go / WASM</text>
  <text x="455" y="15" text-anchor="middle" fill="currentColor" fill-opacity="0.4" font-size="11" font-weight="500">Browser</text>
  <rect x="10" y="24" width="180" height="38" rx="6" fill="#6C8CFF" fill-opacity="0.1" stroke="#6C8CFF" stroke-width="1.2"/>
  <text x="100" y="47" text-anchor="middle" fill="currentColor" font-size="12" font-weight="500">renderer</text>
  <rect x="10" y="70" width="180" height="38" rx="6" fill="#6C8CFF" fill-opacity="0.1" stroke="#6C8CFF" stroke-width="1.2"/>
  <text x="100" y="93" text-anchor="middle" fill="currentColor" font-size="12" font-weight="500">sound</text>
  <rect x="10" y="116" width="180" height="38" rx="6" fill="#6C8CFF" fill-opacity="0.1" stroke="#6C8CFF" stroke-width="1.2"/>
  <text x="100" y="139" text-anchor="middle" fill="currentColor" font-size="12" font-weight="500">run history</text>
  <line x1="205" y1="43" x2="330" y2="43" stroke="currentColor" stroke-opacity="0.3" stroke-width="1.5" marker-end="url(#dg-arr)"/>
  <line x1="205" y1="89" x2="330" y2="89" stroke="currentColor" stroke-opacity="0.3" stroke-width="1.5" marker-end="url(#dg-arr)"/>
  <line x1="205" y1="135" x2="330" y2="135" stroke="currentColor" stroke-opacity="0.3" stroke-width="1.5" marker-start="url(#dg-arr)" marker-end="url(#dg-arr)"/>
  <text x="263" y="62" text-anchor="middle" fill="currentColor" fill-opacity="0.25" font-size="10">syscall/js</text>
  <rect x="345" y="24" width="220" height="38" rx="6" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-opacity="0.2" stroke-width="1"/>
  <text x="455" y="47" text-anchor="middle" fill="currentColor" font-size="12" font-weight="500">Canvas 2D</text>
  <rect x="345" y="70" width="220" height="38" rx="6" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-opacity="0.2" stroke-width="1"/>
  <text x="455" y="93" text-anchor="middle" fill="currentColor" font-size="12" font-weight="500">Web Audio API</text>
  <rect x="345" y="116" width="220" height="38" rx="6" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-opacity="0.2" stroke-width="1"/>
  <text x="455" y="139" text-anchor="middle" fill="currentColor" font-size="12" font-weight="500">localStorage</text>
</svg></figure>
</div>

No external libraries, about 6,600 lines of standard library Go. The WASM compilation step is one command, and the rest is just regular Go.

---

I initially wanted it to be playable on mobile too, but the touch controls complicated the game design more than they were worth, so I cut it. In total it took about a week of evenings, but at some point I dropped it completely and forgot about it for a month. Eventually decided it was worth getting to the end, came back, polished what was left, and shipped it.

It's not well balanced. I was the only playtester, which means I tuned everything around how I play. Some class and gear combinations are probably too strong. Some floors will end a run on turn three for no good reason.

---

What I'm happiest with is the complexity that comes out of fairly simple systems.

<div class="diagram-scroll">
<figure class="center" style="width:100%;"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 580 200" width="100%" aria-label="Dungeon game systems: 8 classes, ~40 gear items, 5 synergies, 36 events, 4 special room types, 3 bosses">
  <rect x="16" y="10" width="175" height="82" rx="8" fill="currentColor" fill-opacity="0.04" stroke="currentColor" stroke-opacity="0.15" stroke-width="1"/>
  <text x="103" y="44" text-anchor="middle" fill="#6C8CFF" font-size="26" font-weight="700">8</text>
  <text x="103" y="62" text-anchor="middle" fill="currentColor" font-size="12" font-weight="600">Classes</text>
  <text x="103" y="78" text-anchor="middle" fill="currentColor" fill-opacity="0.4" font-size="10">4 base · 4 unlockable</text>
  <rect x="203" y="10" width="175" height="82" rx="8" fill="currentColor" fill-opacity="0.04" stroke="currentColor" stroke-opacity="0.15" stroke-width="1"/>
  <text x="290" y="44" text-anchor="middle" fill="#6C8CFF" font-size="26" font-weight="700">~40</text>
  <text x="290" y="62" text-anchor="middle" fill="currentColor" font-size="12" font-weight="600">Gear Items</text>
  <text x="290" y="78" text-anchor="middle" fill="currentColor" fill-opacity="0.4" font-size="10">3 slots · 4 rarities + cursed</text>
  <rect x="390" y="10" width="175" height="82" rx="8" fill="currentColor" fill-opacity="0.04" stroke="currentColor" stroke-opacity="0.15" stroke-width="1"/>
  <text x="477" y="44" text-anchor="middle" fill="#6C8CFF" font-size="26" font-weight="700">5</text>
  <text x="477" y="62" text-anchor="middle" fill="currentColor" font-size="12" font-weight="600">Synergies</text>
  <text x="477" y="78" text-anchor="middle" fill="currentColor" fill-opacity="0.4" font-size="10">item pair bonuses</text>
  <rect x="16" y="106" width="175" height="82" rx="8" fill="currentColor" fill-opacity="0.04" stroke="currentColor" stroke-opacity="0.15" stroke-width="1"/>
  <text x="103" y="140" text-anchor="middle" fill="#6C8CFF" font-size="26" font-weight="700">36</text>
  <text x="103" y="158" text-anchor="middle" fill="currentColor" font-size="12" font-weight="600">Events</text>
  <text x="103" y="174" text-anchor="middle" fill="currentColor" fill-opacity="0.4" font-size="10">random per-floor encounters</text>
  <rect x="203" y="106" width="175" height="82" rx="8" fill="currentColor" fill-opacity="0.04" stroke="currentColor" stroke-opacity="0.15" stroke-width="1"/>
  <text x="290" y="140" text-anchor="middle" fill="#6C8CFF" font-size="26" font-weight="700">4</text>
  <text x="290" y="158" text-anchor="middle" fill="currentColor" font-size="12" font-weight="600">Special Rooms</text>
  <text x="290" y="174" text-anchor="middle" fill="currentColor" fill-opacity="0.4" font-size="10">one per floor, randomly picked</text>
  <rect x="390" y="106" width="175" height="82" rx="8" fill="currentColor" fill-opacity="0.04" stroke="currentColor" stroke-opacity="0.15" stroke-width="1"/>
  <text x="477" y="140" text-anchor="middle" fill="#6C8CFF" font-size="26" font-weight="700">3</text>
  <text x="477" y="158" text-anchor="middle" fill="currentColor" font-size="12" font-weight="600">Bosses</text>
  <text x="477" y="174" text-anchor="middle" fill="currentColor" fill-opacity="0.4" font-size="10">each has a phase 2</text>
</svg></figure>
</div>

Each addition multiplies the number of viable builds in ways I didn't fully predict when adding individual pieces. Watching a run come together around an unplanned combination is exactly why the genre works, and it's satisfying to see it happen even at this scale.

---

Making even a small game gives you a better sense of why certain mechanics work the way they do, what tradeoffs the developers were making, and where the limitations come from. I'd like to make more and explore different genres at some point. We'll see if I get to it.

[Play it here.](https://dungeon.mamonas.dev) Runs in the browser, no install needed.
