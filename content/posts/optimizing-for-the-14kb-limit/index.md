+++ 
title = "Optimizing for the 14KB Limit" 
date = "2025-09-03T08:14:08+03:00" 
tags = ["web", "performance"] 
description = """
Took on the challenge of shrinking my site as much as possible. From trimming scripts to rethinking images, I learned a lot about where the real weight comes from, and how far optimization can go.
"""
+++

I recently watched a [video](https://www.youtube.com/watch?v=ciNXbR5wvhU) going through [this article](https://endtimes.dev/why-your-website-should-be-under-14kb-in-size/) (thanks attention economy). 

The core idea of the article is that when a site is being loaded for the first time, TCP sends 10 packets, to try and figure out how fast the requests can go, increasing the number of packets with each request. The size of these first 10 packets ends up at 14kb, so if we want to make sure the user gets a smooth experience  we should fit our pages within that or at least fit our most important bits.

I find it fun optimizing stuff, so I thought I should go through this exercise and see how far I can push my own site. I’ll go through the strategies I found helpful and the results of my attempt.

---

## Me being a dumdum

The first thing I noticed after opening the network tab was that I was calling Google Analytics twice.

When setting up I added Google Analytics to my template, but it so happened that my theme already included it by default. Removing the extra call cut out ~150 KB right away. Not a huge win in absolute terms, but that’s already more than 15x the "allowed" amount.

---

## Images: The Main Culprit

Something that might be obvious to seasoned web devs but I didn’t even think about - don’t include full-size .png images and .gifs if you don’t need them at that resolution.

I started with reducing image resolutions to only what’s needed (e.g., 1200×1200 to 400×400) and converting images to .webp. 

It took a few attempts to get the result right, as with the size reduction, the images lost some of the sharpness and became blurry, but tuning to keep 100% quality instead of defaults and changing the algorithm used I was able to get pretty good results at a significant size decrease.

My [previous post](https://mamonas.dev/posts/building-a-generative-art-system-in-go/) was 15.25 MB before optimizations. Changing the size and converting to .webp dropped the size to 5.74MB.

<div class="image-row">
  {{< image src="before_image.png" alt="Before Image Conversion" width="600" renderWidth="400">}}
  {{< image src="after_image.png" alt="After Image Conversion" width="600" renderWidth="400">}}
</div>


---

## Converting GIFs to MP4

The gif in the post was still a problem, using 5.16 MB by itself.

I tested a few ffmpeg conversions to shrink it:

```bash
ffmpeg -i blackhole_new.gif \
-movflags faststart -pix_fmt yuv420p -vf "scale=400:-2" blackhole_new.mp4
```

This gave me good size reduction without hurting quality.

I tried adding and adjusting the `-crf` flag to 30 or 26 (default is 23), but the results came with too much of a quality drop, even though it could have significantly reduced the video size I decided to go with the first option.

<div class="image-row">
  {{< image src="gif_reduction.png" alt="GIF size reduction" width="800" renderWidth="600" >}}
</div>

With the image and gif changes, total page size got reduced to 1.8 MB instead of the initial 15.25 MB, still way more than I hoped for, but we will see what we can do about that later.

---

## Theme Optimizations

Next, I decided to slim down my [Hugo](https://gohugo.io/) theme, by cutting off some parts that I did not really care about.

First, I removed the custom font that was used and replaced it with defaults, minus 97KB.

<div class="image-row">
  {{< image src="font_removed.png" alt="font size" width="800" renderWidth="600" >}}
</div>

Second, removed Prism.js, it was used for code block styling. Prism also caused a flash of unstyled code blocks on slow connections, while its only real benefit was a copy button. Not worth the cost. By replacing styling with pure css I reduced my .js file from 178 KB to 907 bytes. **Huge**.


<div class="image-row">
  {{< image src="prisma_size.png" alt="prisma size" width="800" renderWidth="600" >}}
</div>

<br>

<div class="image-row">
  {{< image src="prisma_reduction.png" alt="prisma size 2" width="800" renderWidth="600" >}}
</div>
<br>

---

### Minification

HTML, CSS and JS all contain newline characters, comments and other stuff, that as it happens is not really used when rendering web pages.

Hugo allows to easily minify them by changing a few lines of code in the config and within templates where they are used.

JS was already minimized and minimizing CSS and HTML gave me around another 10 KB, again not that much, but that's about two thirds of what we want to have.

---

## Lazy loading

Another thing that helped with the initial rendering of pages was enabling lazy loading.

This made it so images are only requested when the user scrolls close to them. 

With that all the assets images/videos are not requested initially and we can shed a lot of the data sent with the initial request.

---

## Moving to Cloudflare

Finally I moved hosting from GitHub Pages to Cloudflare. Mainly because of the CDN (content delivery network) that they offer. This gave me a few things:

* Edge caching for faster connections globally.
* Automatic compression with Brotli/gzip for static resources *(spoiler, my whole site is static)*.

---

## Results

I was honestly surprised with the results:

* Home page request reduced to 12.59 KB. **Win.**

<div class="image-row">
  {{< image src="home_page.png" alt="home page" width="800" renderWidth="600" >}}
</div>

* Previous post went from 15 MB to 17.38 KB on the initial load and 600 KB with every asset loaded. **I'll call it a win.**

<div class="image-row">
  {{< image src="generative_art.png" alt="generative art" width="800" renderWidth="600" >}}
</div>

Even though I didn’t quite hit the 14 KB mark with the post, getting it down to 17 KB from 15 MB still felt satisfying. It was a pretty interesting experience trying to shave off just a few more KBs, and I’m sure there are still places I could push this further, but I’ll leave that for another day.

Makes you wonder though. Just how much bandwidth is wasted on the internet every day.
