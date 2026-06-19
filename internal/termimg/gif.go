package termimg

import (
	"image"
	"image/color"
	"image/gif"
	"io"
)

// Frame is one step of an animated walkthrough: the ANSI screen to show and how
// long to hold it, in centiseconds (1/100 s).
type Frame struct {
	ANSI  string
	Delay int
}

// GIF renders every frame and writes an animated GIF. Consecutive frames share
// one median-cut palette; unchanged pixels are emitted as transparent over a
// non-disposed previous frame, which keeps holds and small edits (typing,
// switching tabs) cheap.
func (r *Renderer) GIF(w io.Writer, frames []Frame) error {
	imgs := make([]*image.RGBA, len(frames))
	maxW, maxH := 0, 0
	for i, f := range frames {
		imgs[i] = r.Image(f.ANSI)
		maxW = max(maxW, imgs[i].Bounds().Dx())
		maxH = max(maxH, imgs[i].Bounds().Dy())
	}
	for i := range imgs {
		if imgs[i].Bounds().Dx() != maxW || imgs[i].Bounds().Dy() != maxH {
			imgs[i] = pad(imgs[i], maxW, maxH, r.opt.DefaultBg)
		}
	}

	pal := buildPalette(imgs, 255)
	idx := newIndexer(pal)

	out := &gif.GIF{
		Config:    image.Config{ColorModel: pal, Width: maxW, Height: maxH},
		LoopCount: 0,
	}
	rect := image.Rect(0, 0, maxW, maxH)
	for i, src := range imgs {
		p := image.NewPaletted(rect, pal)
		if i == 0 {
			for j := 0; j+3 < len(src.Pix); j += 4 {
				p.Pix[j/4] = idx.at(src.Pix[j], src.Pix[j+1], src.Pix[j+2])
			}
		} else {
			prev := imgs[i-1].Pix
			cur := src.Pix
			for j := 0; j+3 < len(cur); j += 4 {
				if cur[j] == prev[j] && cur[j+1] == prev[j+1] && cur[j+2] == prev[j+2] {
					p.Pix[j/4] = 0 // transparent: keep the previous frame's pixel
					continue
				}
				p.Pix[j/4] = idx.at(cur[j], cur[j+1], cur[j+2])
			}
		}
		out.Image = append(out.Image, p)
		out.Delay = append(out.Delay, frames[i].Delay)
		out.Disposal = append(out.Disposal, gif.DisposalNone)
	}
	return gif.EncodeAll(w, out)
}

func pad(src *image.RGBA, w, h int, bg color.RGBA) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i+3 < len(dst.Pix); i += 4 {
		dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2], dst.Pix[i+3] = bg.R, bg.G, bg.B, 0xff
	}
	b := src.Bounds()
	for y := 0; y < b.Dy(); y++ {
		copy(dst.Pix[y*dst.Stride:y*dst.Stride+b.Dx()*4], src.Pix[y*src.Stride:y*src.Stride+b.Dx()*4])
	}
	return dst
}

// indexer maps RGB to the nearest opaque palette entry, with a cache.
type indexer struct {
	pal   color.Palette
	cache map[uint32]uint8
}

func newIndexer(pal color.Palette) *indexer {
	return &indexer{pal: pal, cache: map[uint32]uint8{}}
}

func (ix *indexer) at(r, g, b uint8) uint8 {
	key := uint32(r)<<16 | uint32(g)<<8 | uint32(b)
	if v, ok := ix.cache[key]; ok {
		return v
	}
	best, bestD := uint8(1), 1<<31
	for i := 1; i < len(ix.pal); i++ { // skip index 0 (transparent)
		c := ix.pal[i].(color.RGBA)
		dr, dg, db := int(r)-int(c.R), int(g)-int(c.G), int(b)-int(c.B)
		d := dr*dr + dg*dg + db*db
		if d < bestD {
			bestD, best = d, uint8(i)
			if d == 0 {
				break
			}
		}
	}
	ix.cache[key] = best
	return best
}
