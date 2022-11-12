package main

import "github.com/lucagez/patman"

func init() {
	patman.Register("lmao", func(line, arg string) string {
		if line == "lmao" {
			return "this ain't a FUCKING LIEEEEEE"
		}
		return line
	})
}

func main() {
	patman.Run()
}
