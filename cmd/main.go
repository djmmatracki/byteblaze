package main

import (
	"github.com/djmmatracki/byteblaze/internal/app"
	"github.com/sirupsen/logrus"
)

func main() {
	// TODO - add configuration here
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	app.Run(logger, "0.0.0.0", 6881)
}
