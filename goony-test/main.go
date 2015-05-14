package main

import (
	"bufio"
	"fmt"
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
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "goony connectability test")
		})
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	servent := winny.Servent{
		Speed: 10000,
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
		// ch, _ := servent.Search(".jpg")
		ch, _ := servent.Search("")
		cnt := 0
		for key := range ch {
			// log.Printf("File: %s\n", maskKeyword(key.FileName))
			// log.Printf("%d File: %s\n", cnt, key.FileName)
			log.Printf("File: %s\n", key.FileName)
			cnt++
		}
	}()

	go func() {
		ch, _ := servent.KeywordStream()
		for kw := range ch {
			// log.Printf("Search: %s\n", maskKeyword(kw))
			log.Printf("Search: %s\n", kw)
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
	limit := 1
	if len(ru) <= limit {
		return s
	} else {
		return string(ru[0:limit]) + strings.Repeat("*", len(string(ru[limit:])))
	}
}
