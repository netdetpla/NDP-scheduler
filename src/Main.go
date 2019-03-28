package main

import "fmt"

func main()  {
	c := GetConfig()
	fmt.Printf("%s", c.Database)
}
