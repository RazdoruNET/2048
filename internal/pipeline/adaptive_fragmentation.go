package pipeline

import (
	"math/rand"
	"time"
)

type AdaptiveFragmentationModifier struct {
	MinSize        int
	MaxSize        int
	RandomSizes    bool
	MLOptimization bool
	Strategy       string
	rand           *rand.Rand
	lastAdaptation time.Time
}

// NewAdaptiveFragmentationModifier creates new adaptive fragmentation modifier
func NewAdaptiveFragmentationModifier(minSize, maxSize int, randomSizes bool) *AdaptiveFragmentationModifier {
	return &AdaptiveFragmentationModifier{
		MinSize:        minSize,
		MaxSize:        maxSize,
		RandomSizes:    randomSizes,
		MLOptimization: false,
		Strategy:       "simple",
		rand:           rand.New(rand.NewSource(time.Now().UnixNano())),
		lastAdaptation: time.Now(),
	}
}

func (a *AdaptiveFragmentationModifier) Name() string {
	return "adaptive_fragmentation"
}

func (a *AdaptiveFragmentationModifier) Configure(config map[string]interface{}) error {
	if minSize, ok := config["min_size"].(int); ok {
		a.MinSize = minSize
	} else {
		a.MinSize = 64
	}

	if maxSize, ok := config["max_size"].(int); ok {
		a.MaxSize = maxSize
	} else {
		a.MaxSize = 256
	}

	if randomSizes, ok := config["random"].(bool); ok {
		a.RandomSizes = randomSizes
	} else {
		a.RandomSizes = true
	}

	if mlOptimization, ok := config["ml_optimization"].(bool); ok {
		a.MLOptimization = mlOptimization
	} else {
		a.MLOptimization = false
	}

	if strategy, ok := config["strategy"].(string); ok {
		a.Strategy = strategy
	} else {
		a.Strategy = "default"
	}

	a.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	a.lastAdaptation = time.Now()

	return nil
}

func (a *AdaptiveFragmentationModifier) Process(data []byte, direction Direction) []byte {
	if len(data) <= a.MinSize {
		return data
	}

	// For inbound traffic, we don't fragment
	if direction == DirectionInbound {
		return data
	}

	// Fragment outbound traffic with adaptive sizing
	fragmentSizes := a.getAdaptiveFragmentSizes(len(data))

	var result []byte
	for i, size := range fragmentSizes {
		start := i * size
		if start >= len(data) {
			break
		}

		end := start + size
		if end > len(data) {
			end = len(data)
		}

		result = append(result, data[start:end]...)
	}

	return result
}

func (a *AdaptiveFragmentationModifier) getAdaptiveFragmentSizes(dataLen int) []int {
	// Use ML to determine optimal fragmentation
	baseSize := dataLen / 3 // Default to 3 fragments

	if a.RandomSizes {
		// Randomize around base size
		variation := baseSize / 4
		sizes := make([]int, 0, 3)

		for i := 0; i < 3; i++ {
			size := baseSize + a.rand.Intn(variation*2) - variation
			if size < a.MinSize {
				size = a.MinSize
			}
			if size > a.MaxSize {
				size = a.MaxSize
			}
			sizes = append(sizes, size)
		}

		return sizes
	}

	// Fixed size fragmentation
	return []int{a.MinSize, a.MinSize, dataLen - 2*a.MinSize}
}
