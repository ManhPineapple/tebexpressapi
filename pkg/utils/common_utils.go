package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cast"
)

// HandleSigterm -- Handles Ctrl+C or most other means of "controlled" shutdown gracefully.
// Invokes the supplied func before exiting.
func HandleSigterm(handleExit func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		handleExit()
		os.Exit(1)
	}()
}

func ConvertInterfaceToMap(params interface{}) map[string]interface{} {
	paramsConverted := map[string]interface{}{}
	value := reflect.ValueOf(params)

	if value.Kind() == reflect.Map && value.Len() > 0 {
		for _, key := range value.MapKeys() {
			strct := value.MapIndex(key)

			paramskey := fmt.Sprint(key.Interface())
			paramsConverted[paramskey] = strct.Interface()
		}
	}

	return paramsConverted
}

// result: map[string]interface{}, and result convert with bool type
func ConvertInterfaceToMapStringInterface(params interface{}) (map[string]interface{}, bool) {
	s := reflect.ValueOf(params)
	if s.Kind() != reflect.Map {
		return map[string]interface{}{}, false
	}

	value := reflect.ValueOf(params)
	paramsConverted := map[string]interface{}{}
	if value.Len() > 0 {
		for _, key := range value.MapKeys() {
			strct := value.MapIndex(key)

			paramskey := fmt.Sprint(key.Interface())
			paramsConverted[paramskey] = strct.Interface()
		}
	}

	return paramsConverted, true
}

func ConvertInterfaceSliceToMapString(slice interface{}) map[string]interface{} {
	paramsConverted := map[string]interface{}{}
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		return paramsConverted
	}

	for i := 0; i < s.Len(); i++ {
		paramsConverted[fmt.Sprint(i)] = s.Index(i).Interface()
	}

	return paramsConverted
}

func ConvertInterfaceToSlice(slice interface{}) ([]interface{}, bool) {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		return nil, false
	}

	ret := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret, true
}
func Subtract(a, b float64, degits int) float64 {
	e := math.Pow(10, float64(degits))
	return (math.Round(a*e) - math.Round(b*e)) / e
}

func ContainsString(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

func Ceil(v float64, d int) float64 {
	f1 := big.NewFloat(v)
	f2 := big.NewFloat(math.Pow(10, float64(d)))
	var f3 big.Float
	f3.Mul(f1, f2)
	s := f3.String()
	result, _ := strconv.ParseFloat(s, 64)
	return math.Ceil(result) / math.Pow(10, float64(d))
}

func Floor(v float64, d int) float64 {
	f1 := big.NewFloat(v)
	f2 := big.NewFloat(math.Pow(10, float64(d)))
	var f3 big.Float
	f3.Mul(f1, f2)
	s := f3.String()
	result, _ := strconv.ParseFloat(s, 64)
	return math.Floor(result) / math.Pow(10, float64(d))
}

func Round(v float64, d int) float64 {
	sv := fmt.Sprintf(fmt.Sprintf("%%.%df", d), v)
	cv, _ := strconv.ParseFloat(sv, 64)
	return cv
}

func FormatNumber(v float64) string {
	buf := &bytes.Buffer{}
	if v < 0 {
		buf.Write([]byte{'-'})
		v = 0 - v
	}

	comma := []byte{','}

	parts := strings.Split(strconv.FormatFloat(v, 'f', -1, 64), ".")
	pos := 0
	if len(parts[0])%3 != 0 {
		pos += len(parts[0]) % 3
		buf.WriteString(parts[0][:pos])
		buf.Write(comma)
	}
	for ; pos < len(parts[0]); pos += 3 {
		buf.WriteString(parts[0][pos : pos+3])
		buf.Write(comma)
	}
	buf.Truncate(buf.Len() - 1)

	if len(parts) > 1 {
		buf.Write([]byte{'.'})
		if len(parts[1]) < 2 {
			buf.WriteString(parts[1] + "0")

		} else {
			buf.WriteString(parts[1])

		}
	} else {
		buf.Write([]byte{'.'})
		buf.WriteString("00")
	}

	return buf.String()
}

func RoundNumber(num float64) int {
	return int(num + math.Copysign(0.5, num))
}
func ToFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(RoundNumber(num*output)) / output
}

func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func GenerateRandomString(s int) (string, error) {
	b, err := GenerateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}

func TruncateText(s string, max int) string {
	return s[:max]
}

func ContainsNumber(s []int64, e int64) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func DeepCopy(a, b interface{}) error {
	buf, err := json.Marshal(a)
	if err != nil {
		return err
	}

	json.Unmarshal(buf, b)
	return nil
}

func GenerateLastDigitCode(numGen string) int64 {
	number := cast.ToFloat64(numGen)
	var total float64
	mapRatio := map[int]float64{
		1:  3,
		2:  8,
		3:  7,
		4:  6,
		5:  5,
		6:  4,
		7:  3,
		8:  2,
		9:  1,
		10: 1,
		11: 2,
		12: 3,
		13: 3,
		14: 2,
		15: 1,
	}
	for i := 1; i <= 15; i++ {
		mod := math.Mod(number, 10)
		total += mod * mapRatio[i]
		number = (number - mod) / 10
	}

	return cast.ToInt64(math.Mod(cast.ToFloat64(total), 10))
}

func BeginningOfMonth(date time.Time) time.Time {
	return date.AddDate(0, 0, -date.Day()+1)
}

func EndOfMonth(date time.Time) time.Time {
	return date.AddDate(0, 1, -date.Day())
}
