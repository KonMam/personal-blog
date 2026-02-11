+++
title = "Extending a Generative Art System in Go" 
date = "2025-09-17T09:00:11+03:00"
tags = ["go", "generative art"]
categories = ["tech"]
description = """
A follow-up on building a generative art system in Go, showcasing new engines like Perlin Pearls, Swirl, and Cells, along with expanded color palettes to create richer, more varied generative artworks.
"""
images = ["swirl5.png"]
+++

This post continues from [Building a Generative Art System in Go](https://mamonas.dev/posts/building-a-generative-art-system-in-go/), where I build the foundation for a modular generative art framework. If you haven’t read that one, I would suggest starting there.

Here, I will go through what features I have added since, but will mainly focus on showcasing some of the new engines I have implemented.

## Perlin Pearls

The first thing I wanted to add after last time was a new engine, I focused on another engine from [jdxyw/generativeart](https://github.com/jdxyw/generativeart/) called Perlin Pearls. I thought it looked neat and wanted to try my hand at it.

<div class="image-row">
  {{< image src="perlin-perls-jdxyw.png" alt="Perlin Pearls JDXYW" width="600" renderWidth="400" >}}
</div>

At a high level, the algorithm works like this:
- Creates a number of circles and places them randomly on the canvas, making sure they don’t overlap.
- Sprinkles dots around the edge of each circle.
- Uses Perlin noise to move those dots in flowing, organic patterns.
- Tracks the paths of the dots as they move, drawing lines from each dot’s previous position to its new one.
- Adds color variation to the lines, based on the noise values, within a chosen color range.
- Repeats the process many times, layering lines and movements, which builds up textured, pearl-like patterns inside the circles.

My implementation can be found [here](https://github.com/KonMam/go-genart), it mostly follows the same steps just using my internal packages.

The output below adhered to the rules, but the first results felt flat, technically correct, but artistically underwhelming.

<div class="image-row">
  {{< image src="perlinpearls.png" alt="Perlin Pearls 1" width="600" renderWidth="400" >}}
</div>

### More Color Palettes

After experimenting with Perlin Pearls, I quickly realized that monochromatic palettes weren’t doing the visuals justice. The structures were interesting, but the single hue approach flattened the results. To push the system further, I added support for more complex palettes, specifically *split-complementary* and *analogous*, both drawn directly from color theory.

### Split Complementary

Split complementary palettes are built from a base color plus the two hues adjacent to its opposite on the color wheel. They retain the strong contrast of complementary pairs, but with less harshness.

In code, that means: take the hue of the base color, shift 180° for the complement, then offset ±30°. Here’s how I expressed that:
```go
	h, s, _ := RGBToHSL(base.R, base.G, base.B)
	colors := make([]core.RGBA, n)

	// main hue + split complementary (±30° from opposite)
	hues := []float64{
		h,
		math.Mod(h+0.5-1.0/12.0, 1.0),
		math.Mod(h+0.5+1.0/12.0, 1.0),
	}

	for i := 0; i < n; i++ {
		hue := hues[i%3]

		// push saturation high, keep lightness around mid
		saturation := clamp(s*0.9+(float64(i)/float64(n-1))*0.1, 0.6, 1.0)
		lightness := clamp(0.4+(float64(i)/float64(n-1))*0.3, 0, 1)

		r, g, b := HSLToRGB(hue, saturation, lightness)
		colors[i] = core.RGBA{R: r, G: g, B: b, A: 1}
	}
```

### Analogous

An analogous color palette is created by choosing one main color and then using the colors that sit right next to it on the color wheel. Because these colors are close relatives, they blend together smoothly and create a calm, unified feeling.

To keep things from looking too flat, I varied both lightness and saturation across the generated set:
```go
	h, s, l := RGBToHSL(base.R, base.G, base.B)
	colors := make([]core.RGBA, n)

	hues := []float64{math.Mod(h-1.0/12.0, 1.0), h, math.Mod(h+1.0/12.0, 1.0)}

	for i := 0; i < n; i++ {
		// Cycle through the three main hues
		hue := hues[i%3]

		// Vary lightness and saturation
		lightness := clamp(l*0.3+float64(i)/float64(n-1)*0.7, 0, 1)
		saturation := clamp(s*0.5+float64(i)/float64(n-1)*0.5, 0, 1)

		r, g, b := HSLToRGB(hue, saturation, lightness)
		colors[i] = core.RGBA{R: r, G: g, B: b, A: 1}
	}
```


Split complementary injected energy into the compositions, giving a lot of contrast to the pearls, making them feel more dynamic. Analogous created the smoothest results, often echoing the look of the inspirational image.
<div class="image-row">
  {{< image src="perlinpearls2.png" alt="Perlin Pearls 2" width="600" renderWidth="400" >}}
  {{< image src="perlinpearls3.png" alt="Perlin Pearls 3" width="600" renderWidth="400" >}}
</div>

And to end with this engine, here is my current favorite image using it:
<div class="image-row">
  {{< image src="perlinpearls4.png" alt="Perlin Pearls 4" width="600" renderWidth="400" >}}
</div>

## Swirl

After building Perlin Pearls, I wanted to break away from pure reimplementation and experiment with something original.

The vision was to take the perlin pearls and instead of placing them in a random location, doing a swirl of small pearls starting at the center of the image. To do this, I used the golden angle (about 137.5°) which is a special number found in nature, it spaces things evenly, like sunflower seeds or pinecone spirals.

The rest of the engine stayed largely the same (Perlin-driven dots inside circles), but the new arrangement completely changed the feel.

The first renders surprised me: instead of abstract swirls, the output looked like microscopic Petri dishes. This wasn’t what I had envisioned, but it felt more alive than my original plan, so I leaned into it.
<div class="image-row">
  {{< image src="swirl.png" alt="Swirl" width="600" renderWidth="300" >}}
  {{< image src="swirl2.png" alt="Swirl" width="600" renderWidth="300" >}}
  {{< image src="swirl3.png" alt="Swirl" width="600" renderWidth="300" >}}
</div>

Here are two more results, with a different base color, using monochromatic palette. I really like the output with the black background.
<div class="image-row">
  {{< image src="swirl4.png" alt="Swirl" width="800" renderWidth="400" >}}
  {{< image src="swirl5.png" alt="Swirl" width="800" renderWidth="400" >}}
</div>

## Cells

For the next engine, I wanted to bring some structure to the images so I decided to experiment with a grid based engine.

Conceptually, Cells works much like the previous engines: dots move under the influence of Perlin noise, leaving trails as they go. The difference is in the boundaries: instead of circles, the canvas is divided into a grid of square cells, and each cell contains its own self-contained swarm of lines.

For the first image I did a white on black rendition, 10x10 grid, small number of lines.
<div class="image-row">
  {{< image src="cells.png" alt="Cells" width="600" renderWidth="400" >}}
</div>

For the next one I added some color and increased the grid to 50x50, while keeping the number of lines the same within a cell, but reducing the width of the lines:
<div class="image-row">
  {{< image src="cells3.png" alt="Cells 3" width="600" renderWidth="400" >}}
</div>

As you can see, the grid is clear in both pictures, as the lines do not leave their cells and are started and finished in random places on the edges this results in discontinued lines between the cells even though the overall noise pattern is the same.

One particularly interesting lever I have is the noise scale factor:
* Higher values “zoom in” on the noise field, creating chaotic, tangled patterns.
* Lower values smooth it out, giving flowing, wave-like lines.

Here I reduced the factor from 2-3 that was used in previous images, this causes the noise pattern to look sort of zoomed in.
<div class="image-row">
  {{< image src="cells4.png" alt="Cells 4" width="600" renderWidth="400" >}}
</div>

Here are some other examples. Both of these set line number per cell to 25 and a split complementary color palette. The first one uses a factor of 15 which makes it completely chaotic. While the second one reduces scale to 1.5 giving a structured result.
<div class="image-row">
  {{< image src="cells6.png" alt="Cells 6" width="600" renderWidth="400" >}}
  {{< image src="cells7.png" alt="Cells 7" width="600" renderWidth="400" >}}
</div>

## What's next

Looking back, each engine so far has been about static exploration. Freeze a set of rules, run them, and see what the noise field produces.

Right now, small parameter changes radically alter the output. That makes true animation tricky, because every frame risks looking like a completely new piece. Instead, I want to explore animating the process itself: watching the lines grow, points flow, and noise fields evolve.

Think of it less like rendering finished images and more like revealing the drawing as it happens, similar to the feel of Conway’s Game of Life.
