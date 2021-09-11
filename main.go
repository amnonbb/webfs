package main

import (
	"os"
)

var (
	WebPort = os.Getenv("WEB_PORT")
	FilesPath = os.Getenv("FILES_PATH")
)

func main() {
	a := App{}
	a.Initialize()
	a.Run(":" + WebPort)
}
