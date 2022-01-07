package util

import (
	"log"
	"strings"

	"github.com/xvzc/SpoofDPI/config"
)

func ExtractDomain(message *[]byte) string {
	i := 0
	for ; i < len(*message); i++ {
		if (*message)[i] == '\n' {
			i++
			break
		}
	}

	for ; i < len(*message); i++ {
		if (*message)[i] == ' ' {
			i++
			break
		}
	}

	j := i
	for ; j < len(*message); j++ {
		if (*message)[j] == '\n' {
			break
		}
	}

	domain := strings.Split(string((*message)[i:j]), ":")[0]

	return strings.TrimSpace(domain)
}

func ExtractMethod(message *[]byte) string {
	i := 0
	for ; i < len(*message); i++ {
		if (*message)[i] == ' ' {
			break
		}
	}

	method := strings.TrimSpace(string((*message)[:i]))
	Debug(method)

	return strings.ToUpper(method)
}

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
