+++
title = "Extending a Generative Art System in Go" 
date = "2025-09-16T17:50:11+03:00"
tags = ["go", "generative art"] 
description = """
A follow-up on building a generative art system in Go, showcasing new engines like Perlin Pearls, Swirl, and Cells, along with expanded color palettes to create richer, more varied generative artworks.
"""
+++

This is a follow up to my previous post [Building a Generative Art System in Go](https://mamonas.dev/posts/building-a-generative-art-system-in-go/), if you haven’t read that one I would recommend doing that first to be fully caught up.

In this post I will go through what features I have added since, but will mainly focus on showcasing some of the new engines I have implemented.

## Perlin Pearls

The first thing I wanted to add after last time was a new engine, I focused on another engine from [jdxyw/generativeart](https://github.com/jdxyw/generativeart/) called Perlin Pearls. I thought it looked neat and wanted to try my hand at it.

<div class="image-row">
  {{< image src="perlin-pearls-jdxyw.png" alt="Perlin Pearls JDXYW" width="600" renderWidth="400" >}}
</div>

After examining JDXYW code here is what the engine is supposed to do:
- Creates a number of circles and places them randomly on the canvas, making sure they don’t overlap.
- Sprinkles dots around the edge of each circle.
- Uses Perlin noise to move those dots in flowing, organic patterns.
- Tracks the paths of the dots as they move, drawing lines from each dot’s previous position to its new one.
- Adds color variation to the lines, based on the noise values, within a chosen color range.
- Repeats the process many times, layering lines and movements, which builds up textured, pearl-like patterns inside the circles.

My implementation can be found [here](https://github.com/KonMam/go-genart), it mostly follows the same steps just using my internal packages.

I was able to get the below output with this code. The output correctly followed the rules, however it was lacking.

<div class="image-row">
  {{< image src="perlinpearls.png" alt="Perlin Pearls 1" width="600" renderWidth="400" >}}
</div>

## Enter -> More color palettes.

The monochromatic palettes I implemented last time were no longer enough. I decided to add a few more, namely - split-complementary and analogous.

### Split Complementary

A split complementary color palette also starts with one main color, but instead of going to its exact opposite on the color wheel, it takes the two colors that sit on either side of that opposite. This keeps the sense of contrast that complementary colors give, but it softens the clash, resulting in a balanced yet lively look. For example, pairing blue with red-orange and yellow-orange gives you a split complementary palette.

This can be easily expressed in code:
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

An analogous color palette is created by choosing one main color and then using the colors that sit right next to it on the color wheel. Because these colors are close relatives, they blend together smoothly and create a calm, unified feeling. For example, blue, blue-green, and green together form an analogous palette.

This also can be easily expressed in code:
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

Both of these implementations followed the same contract as previously defined with my monochromatic color palette, this meant I could just change out the palette used in the config file, give it a starting color and how many additional ones to generate and it would spit out an image using the specified palette.

Using split complementary I was surprised how much more interesting it made the result. While using the analogous palette gave me the results that were the most similar to the inspirational image (though I used a different starting color, so it is not one to one).
<div class="image-row">
  {{< image src="perlinpearls2.png" alt="Perlin Pearls 2" width="600" renderWidth="400" >}}
  {{< image src="perlinpearls3.png" alt="Perlin Pearls 3" width="600" renderWidth="400" >}}
</div>

And to end with this engine, here is my current favorite image using this engine:
<div class="image-row">
  {{< image src="perlinpearls4.png" alt="Perlin Pearls 4" width="600" renderWidth="400" >}}
</div>

## Swirl

This is the first engine I have implemented without directly referencing some image from [jdxyw/generativeart](https://github.com/jdxyw/generativeart/). After working with the Perlin Pearls engine I wanted to give my own twist or —— swirl on it.

The vision was to take the perlin pearls and instead of placing them in a random location doing a swirl of small pearls starting at the center of the image. To do this, I used the golden angle (about 137.5°) which is a special number found in nature — it spaces things evenly, like sunflower seeds or pinecone spirals.

Other than that the engine mostly stayed the same, however even with this the results were quite surprising.

Here are results of playing around with the color palettes and the engine.
<div class="image-row">
  {{< image src="swirl.png" alt="Swirl" width="600" renderWidth="300" >}}
  {{< image src="swirl2.png" alt="Swirl" width="600" renderWidth="300" >}}
  {{< image src="swirl3.png" alt="Swirl" width="600" renderWidth="300" >}}
</div>

To me this looked a lot like a Petri dish, which was even better than what I envisioned for this engine so I decided to keep it as is and not iterate away.

Here are two more results, with a different base color, using monochromatic palette.
<div class="image-row">
  {{< image src="swirl4.png" alt="Swirl" width="800" renderWidth="400" >}}
  {{< image src="swirl5.png" alt="Swirl" width="800" renderWidth="400" >}}
</div>

## Cells

For the next engine I wanted to bring more structure to the images so I decided to do a grid based engine.

Functionally the engine works in a similar way as the previous two, the only real difference is that the space is divided into a grid of square cells, and each cell has its own dots swirling around. Instead of circles defining boundaries, the grid is the containment and wrapping unit.

For the first image I did a white on black rendition, 10x10 grid, small number of lines.
<div class="image-row">
  {{< image src="cells.png" alt="Cells" width="600" renderWidth="400" >}}
</div>

For the next one I added some color and increased the grid to 50x50, while keeping the number of lines the same within a cell, but reducing the width of the lines:
<div class="image-row">
  {{< image src="cells3.png" alt="Cells 3" width="600" renderWidth="400" >}}
</div>

As you can see, the grid is clear in both pictures, as the lines do not leave their cells and are started and finished in random places on the edges this results in discontinued lines between the cells even though the overall noise pattern is the same.

This is another rendition where I reduced the factor from 2-3 that was used in previous images, this causes the noise pattern to look sort of zoomed in. A higher factor "zooms in" on the noise, creating more chaotic and detailed patterns. A lower factor results in smoother, more flowing lines.
<div class="image-row">
  {{< image src="cells4.png" alt="Cells 4" width="600" renderWidth="400" >}}
</div>

Here are some other examples. Both of these set line number per cell to 25 and a split complementary color palette. The first one uses a factor of 15 which makes it complete chaos. While the second one reduces scale to 1.5.
<div class="image-row">
  {{< image src="cells6.png" alt="Cells 6" width="600" renderWidth="400" >}}
  {{< image src="cells7.png" alt="Cells 7" width="600" renderWidth="400" >}}
</div>

## What's next

Next I would like to work on an engine that could be meaningfully animated, currently a lot of the images are driven by randomness so even small changes completely change the rendition. It would be interesting to animate the drawing part of the engine to see how all of the lines are drawn, something akin to [Conway's Game of Life](https://playgameoflife.com/).
