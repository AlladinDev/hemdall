package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
)

var codes sync.Map

func generateUniqueNumber(lengthOfCode int) string {
	str := ""
	for {
		num := rand.Intn(10) // Change 100 to your desired range
		str += strconv.Itoa(num)
		if len(str) == lengthOfCode {
			if _, exists := codes.Load(str); exists {
				str = ""
				continue
			}
			return str
		}
	}
}

func main() {

	fmt.Print(generateUniqueNumber(6))

}
