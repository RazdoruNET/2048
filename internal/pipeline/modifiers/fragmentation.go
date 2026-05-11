package modifiers

import (
	"math/rand"
	"time"

	"socks5-dpi-proxy/internal/pipeline"
)

type FragmentationModifier struct {
	MinSize int
	MaxSize int
	Random  bool
	rand    *rand.Rand
}

func (f *FragmentationModifier) Name() string {
	return "fragmentation"
}

func (f *FragmentationModifier) Configure(config map[string]interface{}) error {
	if size, ok := config["size"].(int); ok {
		f.MinSize = size
		f.MaxSize = size
	} else {
		if minSize, ok := config["min_size"].(int); ok {
			f.MinSize = minSize
		} else {
			f.MinSize = 100
		}
		
		if maxSize, ok := config["max_size"].(int); ok {
			f.MaxSize = maxSize
		} else {
			f.MaxSize = 200
		}
	}

	if random, ok := config["random"].(bool); ok {
		f.Random = random
	} else {
		f.Random = true
	}

	f.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	return nil
}

func (f *FragmentationModifier) Process(data []byte, direction pipeline.Direction) []byte {
	if len(data) <= f.MinSize {
		return data
	}

	// For inbound traffic, we don't fragment
	if direction == pipeline.DirectionInbound {
		return data
	}

	// Fragment outbound traffic
	var result []byte
	offset := 0

	for offset < len(data) {
		var chunkSize int
		if f.Random {
			chunkSize = f.MinSize + f.rand.Intn(f.MaxSize-f.MinSize+1)
		} else {
			chunkSize = f.MaxSize
		}

		if offset+chunkSize > len(data) {
			chunkSize = len(data) - offset
		}

		result = append(result, data[offset:offset+chunkSize]...)
		offset += chunkSize
	}

	return result
}
