package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

func main() {
	s := pflag.StringP("string", "s", "", "DNA sequence for tandem-repeats scanner")
	f := pflag.StringP("file", "f", "", "file contain a DNA sequence")
	pflag.Parse()

	var r io.Reader
	if *s != "" {
		r = strings.NewReader(*s)
	}
	if *f == "" {
		fmt.Println("please specify a DNA sequence or file name")
		return
	}

	var err error
	r, err = os.Open(*f)
	if err != nil {
		fmt.Printf("cannot open file: %v\n", err)
		return
	}
	res := findRepeat(r)
	json.NewEncoder(os.Stdout).Encode(res)
}

func findRepeat(rr io.Reader) map[string]int {
	r := bufio.NewReader(rr)
	var b byte
	var err error

	repeatMap := make(map[string]*repeat)
	res := make(map[string]int)
	substringExist := make(map[string]struct{})
	var byteBuffer []byte

	for i := 0; true; i++ {
		b, err = r.ReadByte()
		if err != nil {
			break
		}

		// for each repeat obj, try to check with new byte
		for k := range repeatMap {
			m := repeatMap[k]
			if check := m.scan(b); !check {
				// repeation break, release all ignored string related to this substring
				// and add the repeated substring to result if any.
				if m.Repeat > 0 {
					resKey := fmt.Sprintf("%d-%s", m.Start, m.S)
					res[resKey] = m.Repeat + 1
				}
				for _, s := range m.SuperStrings {
					delete(substringExist, s)
				}
				delete(repeatMap, k)
			}
		}

		// append the buffer
		if len(byteBuffer) < 10 {
			byteBuffer = append(byteBuffer, b)
		} else {
			byteBuffer = append(byteBuffer[1:], b)
		}

		if len(byteBuffer) >= 3 {
			// generate substring
			for j := 3; j <= len(byteBuffer); j++ {
				sub := string(byteBuffer[len(byteBuffer)-j:])
				var superStrings []string
				if _, ok := substringExist[sub]; ok {
					continue
				}
				// ignore any string that is multiple of current substring
				// eg. sub = "AAG", ignore "AAGAAG", "AAGAAGAAG"
				for ss := sub; len(ss) < 10; ss += sub {
					substringExist[ss] = struct{}{}
					superStrings = append(superStrings, ss)
				}
				//put each substring to repeat checker
				repeatMap[sub] = &repeat{
					Start:        i + 1 - len(sub),
					S:            []byte(sub),
					SuperStrings: superStrings,
				}
			}
		}
	}
	for k := range repeatMap {
		m := repeatMap[k]
		if m.Repeat > 0 {
			resKey := fmt.Sprintf("%d-%s", m.Start, m.S)
			res[resKey] = m.Repeat + 1
			for _, s := range m.SuperStrings {
				delete(substringExist, s)
			}
		}
	}
	return res
}

type repeat struct {
	Start        int
	S            byteString
	Repeat       int
	CurIndex     int
	SuperStrings []string
}

func (r *repeat) scan(b byte) bool {
	if b != r.S[r.CurIndex] {
		return false
	}
	r.CurIndex++
	if r.CurIndex >= len(r.S) {
		r.Repeat++
		r.CurIndex = 0
	}
	return true
}

type byteString []byte

func (b byteString) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", string(b))), nil
}
