package utils

import (
	"bufio"
	"encoding/csv"
	"io"
	"log"

	"github.com/ctessum/macreader"
)

func ProcessCsv(r io.Reader, withHeader bool, f func(header []string, idx int64, data []string) error) (bool, error) {

	isValidColumn := true

	var headerProcessed = false
	header := make([]string, 0)
	if !withHeader {
		headerProcessed = true
	}

	br := bufio.NewReader(r)
	firstRune, _, err := br.ReadRune()
	if err != nil {
		if err != io.EOF {
			log.Println("cant read rune: ", err)
			return isValidColumn, err
		}

		return isValidColumn, nil
	}
	if firstRune != '\uFEFF' {
		_ = br.UnreadRune()
	}

	lfReader := macreader.New(br)
	var lines = csv.NewReader(lfReader)
	// save memory
	lines.ReuseRecord = true
	var idx = int64(0)
	for {
		line, err := lines.Read()
		if idx == 0 {
			if len(line) != 16 {
				isValidColumn = false
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("Error during receiving file %v", err)
			continue
		}

		idx = idx + 1
		// clone line to another slice because reuseRecord is true
		// should not use copy() here to save memory
		record := append(line[:0:0], line...)
		if len(record) == 0 {
			continue
		}

		if !headerProcessed {
			headerProcessed = true
			header = record
			continue
		}

		if err = f(header, idx, record); err != nil {
			return isValidColumn, err
		}
	}

	return isValidColumn, nil
}

func WriteCSV(w io.Writer, records [][]string) error {
	var writer = csv.NewWriter(w)
	return writer.WriteAll(records)
}
