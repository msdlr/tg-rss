package main

import "fmt"
import "tg-rss/config"

func main() {
	config.LoadConfig()
	fmt.Println("Hello world!")
}
