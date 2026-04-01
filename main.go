package main

import (
	_ "time/tzdata"
	"openclaw-go/cmd"
)

// main.go 只有一个作用就是执行 cmd

func main(){
	cmd.Execute()
}