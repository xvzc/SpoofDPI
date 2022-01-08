package util

import (
	"log"

	"github.com/xvzc/SpoofDPI/config"
)

func Debug(v ...interface{}) {
	if config.GetConfig().Debug == false {
		return
	}

	log.Println(v...)
}

func BytesToChunks(buf []byte) [][]byte {
	if len(buf) < 1 {
		return [][]byte{buf}
	}

	return [][]byte{buf[:1], buf[1:]}
}
