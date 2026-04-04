package main

import "fmt"

// greet returns a friendly greeting.
func greet(name string) string {
	return fmt.Sprintf("Hello, %s! Welcome to tree-glow.", name)
}

func main() {
	names := []string{"Alice", "Bob", "Charlie"}
	for _, name := range names {
		fmt.Println(greet(name))
	}
}
