package main

import (
	"fmt"
	"github.com/vimcolorschemes/worker/util"
)

var Jobs = []string{"import", "clean", "update"}

func main() {
	var job = "clean"
	if array.Find(Jobs, job) {
		fmt.Println("Yes")
	} else {
		fmt.Println("No")
	}
}
