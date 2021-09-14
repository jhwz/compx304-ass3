package main

import (
	"fmt"
	"strings"
	"time"

	. "github.com/klauspost/cpuid/v2"
)

func main() {
	// Print basic CPU information:
	fmt.Println("Name:", CPU.BrandName)
	fmt.Println("PhysicalCores:", CPU.PhysicalCores)
	fmt.Println("ThreadsPerCore:", CPU.ThreadsPerCore)
	fmt.Println("LogicalCores:", CPU.LogicalCores)
	fmt.Println("Family", CPU.Family, "Model:", CPU.Model, "Vendor ID:", CPU.VendorID)
	fmt.Println("Features:", fmt.Sprintf(strings.Join(CPU.FeatureSet(), ",")))
	fmt.Println("Cacheline bytes:", CPU.CacheLine)
	fmt.Println("L1 Data Cache:", CPU.Cache.L1D, "bytes")
	fmt.Println("L1 Instruction Cache:", CPU.Cache.L1D, "bytes")
	fmt.Println("L2 Cache:", CPU.Cache.L2, "bytes")
	fmt.Println("L3 Cache:", CPU.Cache.L3, "bytes")
	fmt.Println("Frequency", CPU.Hz, "hz")

	// Test if we have these specific features:
	if CPU.Supports(SSE, SSE2) {
		fmt.Println("We have Streaming SIMD 2 Extensions")
	}
	array_access()
}

func array_access() {
	const l = 50_000_000
	// Allocate an array
	arr := make([]int, l)
	for i := 0; i < l; i++ {
		arr[i] = i
	}

	var total time.Duration

	tmp := 0
	for i := 0; i < l; i++ {
		// idx := rand.Int() % l
		idx := i

		start := time.Now()
		tmp = arr[idx]
		total += time.Since(start)
	}
	_ = tmp

	fmt.Println(total / time.Duration(l))
}
