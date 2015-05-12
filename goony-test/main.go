package main

import (
	"bufio"
	"github.com/peryaudo/goony/winny"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	servent := winny.Servent{
		Speed: 1000,
		Port:  4504}

	go func() {
		err := readNoderef(&servent)
		if err != nil {
			log.Fatalln(err)
		}

		for {
			time.Sleep(60 * time.Second)
			writeNoderef(&servent)
		}
	}()

	go func() {
		ch, _ := servent.Search("")
		for key := range ch {
			log.Printf("File: %s\n", maskKeyword(key.FileName))
		}
	}()

	go func() {
		ch, _ := servent.KeywordStream()
		for kw := range ch {
			log.Printf("Keyword: %s\n", maskKeyword(kw))
		}
	}()

	log.Fatalln(servent.ListenAndServe())
}

func readNoderef(servent *winny.Servent) (err error) {
	f, err := os.Open("Noderef.txt")
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	r := bufio.NewReader(f)
	for {
		addr, err := r.ReadString(byte('\n'))
		if err == io.EOF {
			err = nil
			break
		}
		if err != nil {
			log.Println(err)
			break
		}
		servent.AddNode(addr)
	}
	return
}

func writeNoderef(servent *winny.Servent) {
	nodeList := servent.NodeList()

	f, err := os.Create("Noderef.txt")
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	for _, s := range nodeList {
		f.WriteString(s)
		f.WriteString("\n")
	}
}

func maskKeyword(s string) string {
	ru := []rune(s)
	if len(ru) <= 2 {
		return s
	} else {
		return string(ru[0:2]) + strings.Repeat("*", len(string(ru[2:])))
	}
}
