package utils

import (
	"math/rand"
	"time"
)

var rng *rand.Rand

func init() {
	source := rand.NewSource(time.Now().UnixNano())
	rng = rand.New(source)
}

// GetRandomNumber returns a random number between min and max (inclusive)
func GetRandomNumber(min, max int) int {
	return rng.Intn(max-min+1) + min
}

// GetRandomTroopIndex returns a random index from 0 to length-1
func GetRandomTroopIndex(length int) int {
	return rng.Intn(length)
}
