package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/klauspost/cpuid/v2"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

type SizeTicks struct {
	Sizes []int
}

func (t SizeTicks) Ticks(min, max float64) []plot.Tick {
	ticks := make([]plot.Tick, 0, len(t.Sizes))
	for _, size := range t.Sizes {
		ticks = append(ticks, plot.Tick{Value: float64(size), Label: formatBytes(int64(size))})
	}

	return ticks
}

type DurationTicks struct {
}

func (t DurationTicks) Ticks(min, max float64) []plot.Tick {

	ticks := plot.DefaultTicks{}.Ticks(min, max)
	for i := range ticks {
		tick := &ticks[i]
		if tick.Label == "" {
			continue
		}
		value, _ := strconv.ParseFloat(tick.Label, 64)

		tick.Label = time.Duration(value).String()
	}
	return ticks
}

func main() {

	var CPU = cpuid.CPU
	// Print basic CPU information:
	fmt.Println("Cache Line Size:", CPU.CacheLine, "bytes")
	fmt.Println("L2 Cache:", formatBytes(int64(CPU.Cache.L2)))
	fmt.Println("L3 Cache:", formatBytes(int64(CPU.Cache.L3)))

	// cache_lines()
	estimate := estimate_llc_size()
	fmt.Println("Estimated LLC size:", formatBytes(int64(estimate)))
}

func estimate_llc_size() int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Use a random source so we index the array randomly.
	// This should help mess with the prefetching.
	testRange := func(sizes []int) plotter.XYs {
		const iterations = 1024 * 1024 * 16

		xys := make(plotter.XYs, 0, len(sizes))
		for _, size := range sizes {
			fmt.Println("Size:", formatBytes(int64(size)))

			// an int32 is 4 bytes, so we allocate size / 4
			arr := make([]int32, size/4)
			arrLength := len(arr)

			start := time.Now()
			for i := 0; i < iterations; i++ {
				arr[r.Int()%arrLength]++
			}
			elapsed := time.Since(start)

			xys = append(xys, plotter.XY{
				X: float64(size),
				Y: float64(int(elapsed.Nanoseconds()) / iterations), // Take the average
			})
		}
		return xys
	}

	plotXYs := func(filename string, xys plotter.XYs, sizes []int) {
		p := plot.New()
		p.Title.Text = "Average access time for different sized arrays"
		p.X.Label.Text = "Array Size"
		p.Y.Label.Text = "Average Time"
		p.Y.Tick.LineStyle = draw.LineStyle{}
		p.X.Tick.Marker = SizeTicks{Sizes: sizes}
		p.Y.Tick.Marker = DurationTicks{}
		p.X.Scale = plot.LogScale{}

		p.Add(&plotter.Line{
			XYs:       xys,
			LineStyle: plotter.DefaultLineStyle,
		})

		if err := p.Save(30*vg.Centimeter, 20*vg.Centimeter, filename+".png"); err != nil {
			panic(err)
		}
	}

	maxSlope := func(xys plotter.XYs) int {
		// Figure out where the largest step was, just using the time differences don't worry about the X value
		max := 0.0
		maxPos := 0
		for i := 0; i < len(xys)-1; i++ {
			slope := (xys[i+1].Y - xys[i].Y)
			if slope > max {
				max = slope
				maxPos = i
			}
		}
		return maxPos
	}

	// Create sizes for the array
	// Initially use a log scale starting from 1KB going to 64MB
	sizes := []int{1024} // 1KB
	for sizes[len(sizes)-1] < 64*1024*1024 {
		sizes = append(sizes, sizes[len(sizes)-1]*2)
	}

	xys := testRange(sizes)
	plotXYs("first", xys, sizes)

	// Find the range with the largest difference, only regarding Y axis
	maxPos := maxSlope(xys)

	// Create a new sizes array, converging on that range but using a linear scale
	sizeRange := sizes[maxPos+1] - sizes[maxPos]
	const linearSteps = 8
	stepSize := int(sizeRange / linearSteps)
	convergedSizes := make([]int, linearSteps)
	for i := 0; i < linearSteps; i++ {
		convergedSizes[i] = sizes[maxPos] + i*stepSize
	}
	xys = testRange(convergedSizes)
	plotXYs("second", xys, convergedSizes)

	return convergedSizes[maxSlope(xys)+1]
}

func cache_lines() {
	const l = 1024 * 1024 * 32

	arr := make([]int32, l)

	xys := make(plotter.XYs, 128)
	for k := 1; k <= 128; k++ {
		// reset
		for i := 0; i < l; i++ {
			arr[i] = 4
		}

		start := time.Now()
		for i := 0; i < l; i += k {
			arr[i] *= 3
		}
		elapsed := time.Since(start)
		fmt.Println(k, ":", elapsed)
		xys[k-1].X = float64(k)
		xys[k-1].Y = float64(elapsed.Nanoseconds())
	}

	p := plot.New()
	p.Title.Text = "Cache lines measurement"
	p.X.Label.Text = "Difference (K)"
	p.Y.Label.Text = "Time (ns)"

	p.Add(&plotter.Line{
		XYs:       xys,
		LineStyle: plotter.DefaultLineStyle,
	})

	if err := p.Save(20*vg.Centimeter, 20*vg.Centimeter, "caches.png"); err != nil {
		panic(err)
	}
}
