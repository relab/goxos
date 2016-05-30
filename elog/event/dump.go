package event

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
)

func DumpAsTextFile(filename string, events []Event) (err error) {
	b := new(bytes.Buffer)
	w := bufio.NewWriter(b)
	for i, event := range events {
		_, err = w.WriteString(fmt.Sprintf("%2d: %v\n", i, event))
		if err != nil {
			return err
		}
	}
	w.Flush()
	err = ioutil.WriteFile(filename, b.Bytes(), 0644)
	return err
}
