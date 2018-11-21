package main

import (
	"strings"
	"time"
)

func round(x float32) int {
	if x >= 0 {
		return int(x + 0.5)
	}
	return int(x - 0.5)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func frames(d time.Duration) int {
	return int(60 * d / time.Second)
}

func romanNumeral(n int) string {
	if n < 10 {
		return convertDigit(n, "X", "V", "I")
	}
	if n < 100 {
		return convertDigit(n/10, "C", "L", "X") + romanNumeral(n%10)
	}
	if n < 1000 {
		return convertDigit(n/100, "M", "D", "C") + romanNumeral(n%100)
	}
	var result string
	for n >= 1000 {
		result += "M"
		n -= 1000
	}
	return result + romanNumeral(n)
}

func convertDigit(d int, high, mid, low string) string {
	if d < 5 {
		return convertDigitBelow5(d, mid, "", low)
	} else {
		return convertDigitBelow5(d-5, high, mid, low)
	}
}

func convertDigitBelow5(d int, high, mid, low string) string {
	if d == 4 {
		return low + high
	} else {
		return mid + strings.Repeat(low, d)
	}
}
