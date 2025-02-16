// Copyright 2023, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package utilfn

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var HexDigits = []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'}

func GetStrArr(v interface{}, field string) []string {
	if v == nil {
		return nil
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	fieldVal := m[field]
	if fieldVal == nil {
		return nil
	}
	iarr, ok := fieldVal.([]interface{})
	if !ok {
		return nil
	}
	var sarr []string
	for _, iv := range iarr {
		if sv, ok := iv.(string); ok {
			sarr = append(sarr, sv)
		}
	}
	return sarr
}

func GetBool(v interface{}, field string) bool {
	if v == nil {
		return false
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return false
	}
	fieldVal := m[field]
	if fieldVal == nil {
		return false
	}
	bval, ok := fieldVal.(bool)
	if !ok {
		return false
	}
	return bval
}

var needsQuoteRe = regexp.MustCompile(`[^\w@%:,./=+-]`)

// minimum maxlen=6
func ShellQuote(val string, forceQuote bool, maxLen int) string {
	if maxLen < 6 {
		maxLen = 6
	}
	rtn := val
	if needsQuoteRe.MatchString(val) {
		rtn = "'" + strings.ReplaceAll(val, "'", `'"'"'`) + "'"
	}
	if strings.HasPrefix(rtn, "\"") || strings.HasPrefix(rtn, "'") {
		if len(rtn) > maxLen {
			return rtn[0:maxLen-4] + "..." + rtn[0:1]
		}
		return rtn
	}
	if forceQuote {
		if len(rtn) > maxLen-2 {
			return "\"" + rtn[0:maxLen-5] + "...\""
		}
		return "\"" + rtn + "\""
	} else {
		if len(rtn) > maxLen {
			return rtn[0:maxLen-3] + "..."
		}
		return rtn
	}
}

func EllipsisStr(s string, maxLen int) string {
	if maxLen < 4 {
		maxLen = 4
	}
	if len(s) > maxLen {
		return s[0:maxLen-3] + "..."
	}
	return s
}

func LongestPrefix(root string, strs []string) string {
	if len(strs) == 0 {
		return root
	}
	if len(strs) == 1 {
		comp := strs[0]
		if len(comp) >= len(root) && strings.HasPrefix(comp, root) {
			if strings.HasSuffix(comp, "/") {
				return strs[0]
			}
			return strs[0]
		}
	}
	lcp := strs[0]
	for i := 1; i < len(strs); i++ {
		s := strs[i]
		for j := 0; j < len(lcp); j++ {
			if j >= len(s) || lcp[j] != s[j] {
				lcp = lcp[0:j]
				break
			}
		}
	}
	if len(lcp) < len(root) || !strings.HasPrefix(lcp, root) {
		return root
	}
	return lcp
}

func ContainsStr(strs []string, test string) bool {
	for _, s := range strs {
		if s == test {
			return true
		}
	}
	return false
}

func IsPrefix(strs []string, test string) bool {
	for _, s := range strs {
		if len(s) > len(test) && strings.HasPrefix(s, test) {
			return true
		}
	}
	return false
}

// sentinel value for StrWithPos.Pos to indicate no position
const NoStrPos = -1

type StrWithPos struct {
	Str string `json:"str"`
	Pos int    `json:"pos"` // this is a 'rune' position (not a byte position)
}

func (sp StrWithPos) String() string {
	return strWithCursor(sp.Str, sp.Pos)
}

func ParseToSP(s string) StrWithPos {
	idx := strings.Index(s, "[*]")
	if idx == -1 {
		return StrWithPos{Str: s, Pos: NoStrPos}
	}
	return StrWithPos{Str: s[0:idx] + s[idx+3:], Pos: utf8.RuneCountInString(s[0:idx])}
}

func strWithCursor(str string, pos int) string {
	if pos == NoStrPos {
		return str
	}
	if pos < 0 {
		// invalid position
		return "[*]_" + str
	}
	if pos > len(str) {
		// invalid position
		return str + "_[*]"
	}
	if pos == len(str) {
		return str + "[*]"
	}
	var rtn []rune
	for _, ch := range str {
		if len(rtn) == pos {
			rtn = append(rtn, '[', '*', ']')
		}
		rtn = append(rtn, ch)
	}
	return string(rtn)
}

func (sp StrWithPos) Prepend(str string) StrWithPos {
	return StrWithPos{Str: str + sp.Str, Pos: utf8.RuneCountInString(str) + sp.Pos}
}

func (sp StrWithPos) Append(str string) StrWithPos {
	return StrWithPos{Str: sp.Str + str, Pos: sp.Pos}
}

// returns base64 hash of data
func Sha1Hash(data []byte) string {
	hvalRaw := sha1.Sum(data)
	hval := base64.StdEncoding.EncodeToString(hvalRaw[:])
	return hval
}

func ChunkSlice[T any](s []T, chunkSize int) [][]T {
	var rtn [][]T
	for len(rtn) > 0 {
		if len(s) <= chunkSize {
			rtn = append(rtn, s)
			break
		}
		rtn = append(rtn, s[:chunkSize])
		s = s[chunkSize:]
	}
	return rtn
}

var ErrOverflow = errors.New("integer overflow")

// Add two int values, returning an error if the result overflows.
func AddInt(left, right int) (int, error) {
	if right > 0 {
		if left > math.MaxInt-right {
			return 0, ErrOverflow
		}
	} else {
		if left < math.MinInt-right {
			return 0, ErrOverflow
		}
	}
	return left + right, nil
}

// Add a slice of ints, returning an error if the result overflows.
func AddIntSlice(vals ...int) (int, error) {
	var rtn int
	for _, v := range vals {
		var err error
		rtn, err = AddInt(rtn, v)
		if err != nil {
			return 0, err
		}
	}
	return rtn, nil
}

func StrsEqual(s1arr []string, s2arr []string) bool {
	if len(s1arr) != len(s2arr) {
		return false
	}
	for i, s1 := range s1arr {
		s2 := s2arr[i]
		if s1 != s2 {
			return false
		}
	}
	return true
}

func StrMapsEqual(m1 map[string]string, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}
	for key, val1 := range m1 {
		val2, found := m2[key]
		if !found || val1 != val2 {
			return false
		}
	}
	for key := range m2 {
		_, found := m1[key]
		if !found {
			return false
		}
	}
	return true
}

func ByteMapsEqual(m1 map[string][]byte, m2 map[string][]byte) bool {
	if len(m1) != len(m2) {
		return false
	}
	for key, val1 := range m1 {
		val2, found := m2[key]
		if !found || !bytes.Equal(val1, val2) {
			return false
		}
	}
	for key := range m2 {
		_, found := m1[key]
		if !found {
			return false
		}
	}
	return true
}

func GetOrderedStringerMapKeys[K interface {
	comparable
	fmt.Stringer
}, V any](m map[K]V) []K {
	keyStrMap := make(map[K]string)
	keys := make([]K, 0, len(m))
	for key := range m {
		keys = append(keys, key)
		keyStrMap[key] = key.String()
	}
	sort.Slice(keys, func(i, j int) bool {
		return keyStrMap[keys[i]] < keyStrMap[keys[j]]
	})
	return keys
}

func GetOrderedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

const (
	nullEncodeEscByte     = '\\'
	nullEncodeSepByte     = '|'
	nullEncodeEqByte      = '='
	nullEncodeZeroByteEsc = '0'
	nullEncodeEscByteEsc  = '\\'
	nullEncodeSepByteEsc  = 's'
	nullEncodeEqByteEsc   = 'e'
)

func EncodeStringMap(m map[string]string) []byte {
	var buf bytes.Buffer
	for idx, key := range GetOrderedMapKeys(m) {
		val := m[key]
		buf.Write(NullEncodeStr(key))
		buf.WriteByte(nullEncodeEqByte)
		buf.Write(NullEncodeStr(val))
		if idx < len(m)-1 {
			buf.WriteByte(nullEncodeSepByte)
		}
	}
	return buf.Bytes()
}

func DecodeStringMap(barr []byte) (map[string]string, error) {
	if len(barr) == 0 {
		return nil, nil
	}
	var rtn = make(map[string]string)
	for _, b := range bytes.Split(barr, []byte{nullEncodeSepByte}) {
		keyVal := bytes.SplitN(b, []byte{nullEncodeEqByte}, 2)
		if len(keyVal) != 2 {
			return nil, fmt.Errorf("invalid null encoding: %s", string(b))
		}
		key, err := NullDecodeStr(keyVal[0])
		if err != nil {
			return nil, err
		}
		val, err := NullDecodeStr(keyVal[1])
		if err != nil {
			return nil, err
		}
		rtn[key] = val
	}
	return rtn, nil
}

func EncodeStringArray(arr []string) []byte {
	var buf bytes.Buffer
	for idx, s := range arr {
		buf.Write(NullEncodeStr(s))
		if idx < len(arr)-1 {
			buf.WriteByte(nullEncodeSepByte)
		}
	}
	return buf.Bytes()
}

func DecodeStringArray(barr []byte) ([]string, error) {
	if len(barr) == 0 {
		return nil, nil
	}
	var rtn []string
	for _, b := range bytes.Split(barr, []byte{nullEncodeSepByte}) {
		s, err := NullDecodeStr(b)
		if err != nil {
			return nil, err
		}
		rtn = append(rtn, s)
	}
	return rtn, nil
}

func EncodedStringArrayHasFirstKey(encoded []byte, firstKey string) bool {
	firstKeyBytes := NullEncodeStr(firstKey)
	if !bytes.HasPrefix(encoded, firstKeyBytes) {
		return false
	}
	if len(encoded) == len(firstKeyBytes) || encoded[len(firstKeyBytes)] == nullEncodeSepByte {
		return true
	}
	return false
}

// encodes a string, removing null/zero bytes (and separators '|')
// a zero byte is encoded as "\0", a '\' is encoded as "\\", sep is encoded as "\s"
// allows for easy double splitting (first on \x00, and next on "|")
func NullEncodeStr(s string) []byte {
	strBytes := []byte(s)
	if bytes.IndexByte(strBytes, 0) == -1 &&
		bytes.IndexByte(strBytes, nullEncodeEscByte) == -1 &&
		bytes.IndexByte(strBytes, nullEncodeSepByte) == -1 &&
		bytes.IndexByte(strBytes, nullEncodeEqByte) == -1 {
		return strBytes
	}
	var rtn []byte
	for _, b := range strBytes {
		if b == 0 {
			rtn = append(rtn, nullEncodeEscByte, nullEncodeZeroByteEsc)
		} else if b == nullEncodeEscByte {
			rtn = append(rtn, nullEncodeEscByte, nullEncodeEscByteEsc)
		} else if b == nullEncodeSepByte {
			rtn = append(rtn, nullEncodeEscByte, nullEncodeSepByteEsc)
		} else if b == nullEncodeEqByte {
			rtn = append(rtn, nullEncodeEscByte, nullEncodeEqByteEsc)
		} else {
			rtn = append(rtn, b)
		}
	}
	return rtn
}

func NullDecodeStr(barr []byte) (string, error) {
	if bytes.IndexByte(barr, nullEncodeEscByte) == -1 {
		return string(barr), nil
	}
	var rtn []byte
	for i := 0; i < len(barr); i++ {
		curByte := barr[i]
		if curByte == nullEncodeEscByte {
			i++
			nextByte := barr[i]
			if nextByte == nullEncodeZeroByteEsc {
				rtn = append(rtn, 0)
			} else if nextByte == nullEncodeEscByteEsc {
				rtn = append(rtn, nullEncodeEscByte)
			} else if nextByte == nullEncodeSepByteEsc {
				rtn = append(rtn, nullEncodeSepByte)
			} else if nextByte == nullEncodeEqByteEsc {
				rtn = append(rtn, nullEncodeEqByte)
			} else {
				// invalid encoding
				return "", fmt.Errorf("invalid null encoding: %d", nextByte)
			}
		} else {
			rtn = append(rtn, curByte)
		}
	}
	return string(rtn), nil
}

func SortStringRunes(s string) string {
	runes := []rune(s)
	sort.Slice(runes, func(i, j int) bool {
		return runes[i] < runes[j]
	})
	return string(runes)
}

// will overwrite m1 with m2's values
func CombineMaps[V any](m1 map[string]V, m2 map[string]V) {
	for key, val := range m2 {
		m1[key] = val
	}
}

// returns hex escaped string (\xNN for each byte)
func ShellHexEscape(s string) string {
	var rtn []byte
	for _, ch := range []byte(s) {
		rtn = append(rtn, []byte(fmt.Sprintf("\\x%02x", ch))...)
	}
	return string(rtn)
}

func GetMapKeys[K comparable, V any](m map[K]V) []K {
	var rtn []K
	for key := range m {
		rtn = append(rtn, key)
	}
	return rtn
}

// combines string arrays and removes duplicates (returns a new array)
func CombineStrArrays(sarr1 []string, sarr2 []string) []string {
	var rtn []string
	m := make(map[string]struct{})
	for _, s := range sarr1 {
		if _, found := m[s]; found {
			continue
		}
		m[s] = struct{}{}
		rtn = append(rtn, s)
	}
	for _, s := range sarr2 {
		if _, found := m[s]; found {
			continue
		}
		m[s] = struct{}{}
		rtn = append(rtn, s)
	}
	return rtn
}
