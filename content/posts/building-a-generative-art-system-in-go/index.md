+++ 
title = "Building a Generative Art System in Go" 
date = "2025-08-26T21:06:08+03:00" 
tags = ["go", "generative art"] 
description = """
Generative art shifts the focus from drawing images to designing systems. Instead of sketching directly, you define algorithms, randomness, and rules, then let the system produce the output. For me what makes it interesting is the fact that you don’t just create one piece, but a machine capable of generating infinite variations.
"""
+++

## 1. What is Generative Art?

Generative art shifts the focus from drawing images to designing systems. Instead of sketching directly, you define algorithms, randomness, and rules, then let the system produce the output. For me what makes it interesting is the fact that you don’t just create one piece, but a machine capable of generating infinite variations.

This post walks through how I approached building such a system in Go, you can find it [here](https://github.com/KonMam/go-genart).

## 2. Architecture and Contracts

One thing I learnt when starting any project is that it is worthwhile to invest time and think beyond the first few steps and try to design it to be enjoyable to work with if I needed to add more functionality. This saves a lot of refactoring time.
To keep engines swappable and avoid tangling geometry, color, and rendering, I sketched out a few well-defined contracts.

**The Engine**
An engine is just an algorithm. It takes randomness and parameters and returns a `Scene`:

```go
type Engine interface {
    Name() string
    Generate(ctx context.Context, rng *rand.Rand, params map[string]float64) (Scene, error)
}
```

**The Scene**
A scene is not pixels but geometry and color information:

```go
type Scene struct {
    Items []Item
}
```

This separation makes rendering independent. The same scene can go to PNG, SVG, PDF, or even a plotter without changing the engine.

**The Palette**
A palette supplies colors. Engines don’t worry about color theory:

```go
type Palette interface {
    Colors() []core.RGBA
    Pick(rng *rand.Rand) core.RGBA
}
```

Engines generate geometry, palettes provide color, renderers handle pixels. Each can evolve independently.

## 3. Geometric Primitives

Before noise and turbulence, I needed fundamentals. The `geom` package holds:

* **Vec2**: a 2D vector type with operations like add, scale, rotate.
* **Shapes**: functions such as `Polygon(cx, cy, r, n)` or `Circle(cx, cy, r)`.
* **Transforms**: reusable translate, rotate, scale helpers.

With these, I could already produce scenes like a single circle or square, rendered cleanly to PNG. Even if it doesn't look too exciting, I was pleased that the approach worked.

<div class="image-row">
  {{< image src="square.png" alt="Square" width="400" >}}
  {{< image src="circle.png" alt="Circle" width="400" >}}
</div>

## 4. Noise

To get organic patterns, you want smooth randomness where nearby coordinates produce similar values instead of pure rng. Noise functions like Simplex or Perlin solve this.
To start with I created a wrapper around [ojrac/opensimplex-go](https://github.com/ojrac/opensimplex-go).

This is what Simplex noise looks like:
<div class="image-row">
  {{< image src="simplex-noise.png" alt="Simplex Noise" width="200" >}}
</div>

I outputted noise in a simple interface, which I can reuse for different Noise algorithms.
```go
type ScalarField2D interface {
    At(x, y float64) float64
}
```

With this abstraction, engines can sample fields without caring how they’re generated. From there:

* Remap values from `[-1,1]` into useful ranges like `[0,1]` for opacity or `[0,360]` for angles.
* Compute gradients to turn noise into vector fields. Walking along gradients produces flow-like motion; walking perpendicular produces contour lines.

Noise is the key ingredient for organic complexity and a lot of algorithms that produce compelling visuals.

## 5. Flow Fields

The first thing I tried generating were `flowfields` which use noise gradients to drive thousands of particles.

How it works:
1. Overlay a grid on the canvas.
2. At each grid point, compute a vector from noise.
3. Drop particles randomly.
4. Move each step using the vector at its location, drawing as it goes.

The output depends on the noise:

* **Flow Waves**: smooth noise -> long sweeping curves.
* **Flow Clouds**: turbulent noise -> chaotic clusters, smoke-like.

Still need to work some more on the parameters to get them looking right, but it's a good start:

<div class="image-row">
  {{< image src="flow_waves.png" alt="Flow Waves" width="600" renderWidth="400" >}}
  {{< image src="flow_clouds.png" alt="Flow Clouds" width="600" renderWidth="400" >}}
</div>

## 6. The Blackhole Engine

Next I decided to try tackling something more ambitious. In [jdxyw/generativeart](https://github.com/jdxyw/generativeart) project I have found a lot of inspiration.

One of those was the **Blackhole**. It was majestic.
<div class="image-row">
  {{< image src="jdxyw_blackhole.png" alt="JDXYW Blackhole" width="400" >}}
</div>

The engine starts with concentric circles, then lets noise distort them into collapsing rings.

**Core idea:**

* Subdivide each circle into angular steps.
* For each step, take the base radius and add a noise-driven offset.
* Clamp to a minimum radius to preserve the central hole.

```go
// -- excerpt --
    theta := startTheta + 2*math.Pi*float64(j)/float64(segments)

    // High-frequency noise
    r1 := math.Cos(theta) + 1
    r2 := math.Sin(theta) + 1
    nv := field.At(k*freq*r1, k*freq*r2, float64(i)*circleGap)

    r := radius + nv*noisiness*amp

    // -- continued --
}
```

Artifacts showed up early: straight bands, jagged edges. 


<div class="image-row">
  {{< image src="artefact_1.png" alt="Artefact" width="200" >}}
  {{< image src="artefact_2.png" alt="Artefact" width="200" >}}
  {{< image src="artefact_3.png" alt="Artefact" width="200" >}}
</div>

Fixes included randomizing the start angle per circle, adding alpha jitter so overlaps blend, and enabling supersampling. 

After some trial and error I managed to get this result:

<div class="image-row">
  {{< image src="first_blackhole.png" alt="First Black Hole" width="400" >}}
</div>

The blackhole is more turbulent and not as smooth/blended as the inspiration, but I think this is what makes generative art so interesting, endless possibilities.

## 7. Monochromatic Palettes

One thing I wasn't happy was the color palette I was using. Throughout testing I added a few of them, but hardcoding them felt limiting.
Enter - **Monochromatic Palettes**.

How it works:
1. Start with a base RGB color.
2. Convert to HSL.
3. Fix the hue.
4. Generate variations by tweaking saturation and lightness.
5. Convert back to RGB.

```go
func Monochrome(base core.RGBA, n int) []core.RGBA {
	if n < 2 {
		n = 2
	}

	h, s, l := RGBToHSL(base.R, base.G, base.B)
	colors := make([]core.RGBA, n)

	for i := 0; i < n; i++ {
		f := float64(i) / float64(n-1)
		// lightness range: darker to lighter around base
		newL := clamp(l*0.3+f*0.7, 0, 1)
		r, g, b := HSLToRGB(h, s, newL)
		colors[i] = core.RGBA{R: r, G: g, B: b, A: 1}
	}
	return colors
}
```

With this all I needed to do was give some base color in the config and the rest would be handled.
As generative art in a lot of cases is about complex geometry, minimal color palettes give another dimension for tweaking without taking away from the core.

<div class="image-row">
  {{< image src="blackhole_mono1.png" alt="Monochromatic Blackhole" width="300" >}}
  {{< image src="blackhole_mono2.png" alt="Monochromatic Blackhole" width="300" >}}
  {{< image src="blackhole_mono3.png" alt="Monochromatic Blackhole" width="300" >}}
</div>

## 8. Config-Driven Runs

As engines gained parameters, CLI flags became messy and I was starting to have a bad time.

```bash
go run ./cmd/genart -engine blackhole \
-palette mono -palette-base "0.25,0.5,0.25" -palette-n 8 \
-params "circles=1000,density=0.2,gap=0.012,lw=0.0003,hole=0.08,freq=3,amp=1" \
-seed 42 -w 1200 -h 1200 \
-bg "0,0,0" \
-out blackhole_green.png
```

Not good. I investigated a few approaches, moving away from `flags` in go standard library, using `yaml` files, but I settled down with good old `json`, mainly because I was already outputting it after runs for logging purposes, but it was also easier to work with.

This meant I could create a config file with the parameters I want, and run it with a simple command.

```json
{
  "engine": "blackhole",
  "seed": 42,
  "width": 1024,
  "height": 1024,
  "palette": {
    "type": "mono",
    "base": [0.2,0.4,0.7,1]
  },
  "params": {
    "amp": 1.2,
    "density": 0.3
  }
}
```

```bash
go run ./cmd/genart -config ./inputs/blackhole_mono3.json
```

Much more elegant. Additionally, since I was also outputing json for logging, I could pipe it out into another file and just plug it back in to get the same results. 

Without the change any further functionality would have required more CLI flags and we were already 5 lines deep into them, so I was happy.

## 9. GIFs (JIFs)

Having opened up the path for more functionality (not scope creep), I decided to tackle something that would bring the project to the next level. Support for animations.

GIF generation slots in naturally as a new orchestration layer:
* A new package internal/anim is responsible for running an engine repeatedly, varying parameters over time, rendering each frame, and combining frames into a GIF.
* main.go simply checks: if the config contains an animation section, it delegates to anim.Run; otherwise it runs a single static render.

Adding animation required only an orchestration layer. Engines and renderers stayed unchanged.

Configs gained an `animation` block in which both parametters and colors could be tweaked over time:

```json
"animation": {
  "duration": 5,
  "fps": 20,
  "easing": "cosine",
  "vary": {
    "amp": [0.9, 1.1],
    "palette.base": [[0.2,0.4,0.7,1],[0.8,0.3,0.2,1]]
  }
}
```

My hope was to generate a smooth pullsing blackhole. 

<div class="image-row">
  {{< lazy-video width="400" height="400" placeholder="blackhole_new_first_frame.webp" mp4="blackhole_new.mp4" >}}
</div>

This is not exactly there yet, the changes to the engine influence the render too quickly, but I think with some tweaking it could get there.

## 10. What’s Next

The base system is in place, I can do static images, I can do animations. The next step is to play around with more engines, try to implement some other interesting ones from [jdxyw/generativeart](https://github.com/jdxyw/generativeart) and tweak them based on my tastes.

Other than that, the project could use additional noise options (Perlin, Worley), more renderer options to add support for SVG and MP4. Maybe a graphical UI to see live changes as parameters are tweaked.
