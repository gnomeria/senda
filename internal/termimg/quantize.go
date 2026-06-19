package termimg

import (
	"image"
	"image/color"
	"sort"
)

// buildPalette derives a single global palette (≤ maxColors opaque entries)
// shared across all frames via median-cut, with index 0 reserved as a fully
// transparent slot for delta-frame encoding.
func buildPalette(frames []*image.RGBA, maxColors int) color.Palette {
	hist := map[color.RGBA]int{}
	for _, f := range frames {
		px := f.Pix
		for i := 0; i+3 < len(px); i += 4 {
			c := color.RGBA{px[i], px[i+1], px[i+2], 0xff}
			hist[c]++
		}
	}
	colors := make([]wcolor, 0, len(hist))
	for c, n := range hist {
		colors = append(colors, wcolor{c, n})
	}

	pal := color.Palette{color.RGBA{0, 0, 0, 0}} // index 0: transparent
	if len(colors) <= maxColors {
		// Few enough distinct colours to use them verbatim — crispest result.
		sort.Slice(colors, func(i, j int) bool { return colors[i].n > colors[j].n })
		for _, c := range colors {
			pal = append(pal, c.c)
		}
		return pal
	}

	for _, b := range medianCut(colors, maxColors) {
		pal = append(pal, b.avg())
	}
	return pal
}

type wcolor struct {
	c color.RGBA
	n int
}

type bucket struct {
	colors []wcolor
}

func (b bucket) count() int {
	n := 0
	for _, c := range b.colors {
		n += c.n
	}
	return n
}

// ranges returns the per-channel spread of the bucket.
func (b bucket) ranges() (r, g, bl int) {
	rMin, gMin, bMin := 255, 255, 255
	rMax, gMax, bMax := 0, 0, 0
	for _, c := range b.colors {
		rMin, rMax = min(rMin, int(c.c.R)), max(rMax, int(c.c.R))
		gMin, gMax = min(gMin, int(c.c.G)), max(gMax, int(c.c.G))
		bMin, bMax = min(bMin, int(c.c.B)), max(bMax, int(c.c.B))
	}
	return rMax - rMin, gMax - gMin, bMax - bMin
}

func (b bucket) avg() color.RGBA {
	var rs, gs, bs, ns int
	for _, c := range b.colors {
		rs += int(c.c.R) * c.n
		gs += int(c.c.G) * c.n
		bs += int(c.c.B) * c.n
		ns += c.n
	}
	if ns == 0 {
		return color.RGBA{0, 0, 0, 0xff}
	}
	return color.RGBA{uint8(rs / ns), uint8(gs / ns), uint8(bs / ns), 0xff}
}

// medianCut splits the colour set into n buckets, repeatedly subdividing the
// bucket with the largest count×spread along its widest channel.
func medianCut(colors []wcolor, n int) []bucket {
	buckets := []bucket{{colors}}
	for len(buckets) < n {
		bi := -1
		bestScore := -1
		for i, b := range buckets {
			if len(b.colors) < 2 {
				continue
			}
			r, g, bl := b.ranges()
			// Split the widest box (largest colour spread), not the one with the
			// most pixels — otherwise a large flat background starves the few
			// saturated accent colours of palette slots and washes them to grey.
			score := max(r, max(g, bl))
			if score > bestScore {
				bestScore, bi = score, i
			}
		}
		if bi < 0 {
			break
		}
		b := buckets[bi]
		r, g, bl := b.ranges()
		ch := 0
		if g >= r && g >= bl {
			ch = 1
		} else if bl >= r && bl >= g {
			ch = 2
		}
		sort.Slice(b.colors, func(i, j int) bool {
			return channel(b.colors[i].c, ch) < channel(b.colors[j].c, ch)
		})
		// Split at the weighted median so both halves carry similar pixel mass.
		half := b.count() / 2
		acc, mid := 0, 1
		for mid = 1; mid < len(b.colors); mid++ {
			acc += b.colors[mid-1].n
			if acc >= half {
				break
			}
		}
		left := bucket{append([]wcolor(nil), b.colors[:mid]...)}
		right := bucket{append([]wcolor(nil), b.colors[mid:]...)}
		buckets[bi] = left
		buckets = append(buckets, right)
	}
	return buckets
}

func channel(c color.RGBA, ch int) uint8 {
	switch ch {
	case 0:
		return c.R
	case 1:
		return c.G
	default:
		return c.B
	}
}
