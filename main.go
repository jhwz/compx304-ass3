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

func main() {

	var CPU = cpuid.CPU
	fmt.Println("CPU Information (similar to lscpu)")
	fmt.Println("Cache Line Size:", CPU.CacheLine, "bytes")
	fmt.Println("L2 Cache:", formatBytes(int64(CPU.Cache.L2)))
	fmt.Println("L3 Cache:", formatBytes(int64(CPU.Cache.L3)))

	fmt.Println("Running estimation algorithm")
	estimate := estimate_llc_size()
	fmt.Println("Estimated LLC size:", formatBytes(int64(estimate)))
}

func estimate_llc_size() int {
	// Use a random source so we index the array randomly.
	// This should help mess with the prefetching.
	// It should also handle the issues with cachelines.
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// This type encompasses an entire cache line (assuming a cacheLineSize of 64 bytes).
	// Using an array of these will force each access to be in a new cacheline.
	type cacheLine64 struct {
		a int64 // 8 bytes
		_ int64 // 16 bytes
		_ int64 // 24 bytes
		_ int64 // 32 bytes
		_ int64 // 40 bytes
		_ int64 // 48 bytes
		_ int64 // 56 bytes
		_ int64 // 64 bytes
	}

	// testRange tests the sizes given and records them in
	// the results array. It accesses random values in the array and
	// increments them so some operation is performed.
	//
	// The resulting array maps each size to the average execution time for each operation
	testRange := func(sizes []int, iterations int) plotter.XYs {

		xys := make(plotter.XYs, 0, len(sizes))
		for _, size := range sizes {
			arr := make([]cacheLine64, size/64)
			arrLength := len(arr)

			start := time.Now()
			for i := 0; i < iterations; i++ {
				arr[r.Int()%arrLength].a++
			}
			elapsed := time.Since(start)
			fmt.Printf("Array size: %-16s Duration: %s\n", formatBytes(int64(size)), elapsed)

			xys = append(xys, plotter.XY{
				X: float64(size),
				Y: float64(int(elapsed.Nanoseconds()) / iterations), // Take the average
			})
		}
		return xys
	}

	// plotXYs is a nice helper function which plots the output from testRange
	// and saves it to a file. Helpful for debugging and seeing what the program is
	// seeing!
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

	// Max slope doesn't actually return the max slope, it returns the index of the point
	// which is steepest
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

	xys := testRange(sizes, 8_000_000)
	plotXYs("step1", xys, sizes)

	// Find the range with the largest difference, only regarding Y axis
	maxPos := maxSlope(xys)

	fmt.Printf("\nConverging on range %s-%s\n", formatBytes(int64(sizes[maxPos])), formatBytes(int64(sizes[maxPos+1])))
	// Create a new sizes array, converging on that range but using a linear scale
	sizeRange := sizes[maxPos+1] - sizes[maxPos]
	const linearSteps = 8
	stepSize := int(sizeRange / linearSteps)
	convergedSizes := make([]int, linearSteps)
	for i := 0; i < linearSteps; i++ {
		convergedSizes[i] = sizes[maxPos] + i*stepSize
	}
	xys = testRange(convergedSizes, 32_000_000)
	plotXYs("step2", xys, convergedSizes)

	return convergedSizes[maxSlope(xys)]
}

// formats the bytes to IEC format
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

// Helper functions for the plotting

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
