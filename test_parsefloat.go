package main

import (
	"fmt"
	"strconv"
)

func parseFloat(s string) float64 {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return 0
}

func main() {
	fmt.Println("parseFloat(\"\"):", parseFloat(""))
	fmt.Println("parseFloat(\"123.45\"):", parseFloat("123.45"))
	fmt.Println("parseFloat(\"invalid\"):", parseFloat("invalid"))
}