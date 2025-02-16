package string_util

import (
	"compress/zlib"
	"encoding/binary"
	"io"
	"strings"
	"sync"
	"unicode"
)

const pooledCapacity = 64

var (
	slicePool        sync.Pool
	decodingOnce     sync.Once
	transliterations [65536][]rune
	transCount       = rune(len(transliterations))
	getUint16        = binary.LittleEndian.Uint16
)

func decodeTransliterations() {
	r, err := zlib.NewReader(strings.NewReader(tableData))
	if err != nil {
		panic(err)
	}
	defer r.Close()
	tmp1 := make([]byte, 2)
	tmp2 := tmp1[:1]
	for {
		if _, err := io.ReadAtLeast(r, tmp1, 2); err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		chr := getUint16(tmp1)
		if _, err := io.ReadAtLeast(r, tmp2, 1); err != nil {
			panic(err)
		}
		b := make([]byte, int(tmp2[0]))
		if _, err := io.ReadFull(r, b); err != nil {
			panic(err)
		}
		transliterations[int(chr)] = []rune(string(b))
	}
}

func Unidecode(s string) string {
	decodingOnce.Do(decodeTransliterations)
	l := len(s)
	var r []rune
	if l > pooledCapacity {
		r = make([]rune, 0, len(s))
	} else {
		if x := slicePool.Get(); x != nil {
			r = x.([]rune)[:0]
		} else {
			r = make([]rune, 0, pooledCapacity)
		}
	}
	for _, c := range s {
		if c <= unicode.MaxASCII {
			r = append(r, c)
			continue
		}
		if c > unicode.MaxRune || c > transCount {
			/* Ignore reserved chars */
			continue
		}
		if d := transliterations[c]; d != nil {
			r = append(r, d...)
		}
	}
	res := string(r)
	if l <= pooledCapacity {
		slicePool.Put(r)
	}
	return res
}
