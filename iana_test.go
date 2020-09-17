// +build !js

package stun

import (
	"bytes"
	"encoding/csv"
	"strconv"
	"strings"
	"testing"
)

func loadCSV(t testing.TB, name string) [][]string {
	t.Helper()
	data := loadData(t, name)
	r := csv.NewReader(bytes.NewReader(data))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return records
}

func TestIANA(t *testing.T) {
	t.Run("Methods", func(t *testing.T) {
		records := loadCSV(t, "stun-parameters-2.csv")
		m := make(map[string]Method)
		for _, r := range records[1:] {
			var (
				v    = r[0]
				name = r[1]
			)
			if strings.Contains(v, "-") {
				continue
			}
			val, parseErr := strconv.ParseInt(v[2:], 16, 64)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			t.Logf("value: 0x%x, name: %s", val, name)
			m[name] = Method(val)
		}
		for val, name := range methodName() {
			mapped, ok := m[name]
			if !ok {
				t.Errorf("failed to find method %s in IANA", name)
			}
			if mapped != val {
				t.Errorf("%s: IANA %d != actual %d", name, mapped, val)
			}
		}
	})
	t.Run("Attributes", func(t *testing.T) {
		records := loadCSV(t, "stun-parameters-4.csv")
		m := map[string]AttrType{}
		for _, r := range records[1:] {
			var (
				v    = r[0]
				name = r[1]
			)
			if strings.Contains(v, "-") {
				continue
			}
			val, parseErr := strconv.ParseInt(v[2:], 16, 64)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			t.Logf("value: 0x%x, name: %s", val, name)
			m[name] = AttrType(val)
		}
		// Not registered in IANA.
		for k, v := range map[string]AttrType{
			"ORIGIN": 0x802F,
		} {
			m[k] = v
		}
		for val, name := range attrNames() {
			mapped, ok := m[name]
			if !ok {
				t.Errorf("failed to find attribute %s in IANA", name)
			}
			if mapped != val {
				t.Errorf("%s: IANA %d != actual %d", name, mapped, val)
			}
		}
	})
	t.Run("ErrorCodes", func(t *testing.T) {
		records := loadCSV(t, "stun-parameters-6.csv")
		m := map[string]ErrorCode{}
		for _, r := range records[1:] {
			var (
				v    = r[0]
				name = r[1]
			)
			if strings.Contains(v, "-") {
				continue
			}
			val, parseErr := strconv.ParseInt(v, 10, 64)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			t.Logf("value: 0x%x, name: %s", val, name)
			m[name] = ErrorCode(val)
		}
		for val, nameB := range errorReasons {
			name := string(nameB)
			mapped, ok := m[name]
			if !ok {
				t.Errorf("failed to find error code %s in IANA", name)
			}
			if mapped != val {
				t.Errorf("%s: IANA %d != actual %d", name, mapped, val)
			}
		}
	})
}
