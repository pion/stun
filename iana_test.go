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
		for val, name := range methodName {
			mapped, ok := m[name]
			if !ok {
				t.Errorf("failed to find method %s in IANA", name)
			}
			if mapped != val {
				t.Errorf("%s: IANA %d != actual %d", name, mapped, val)
			}
		}
	})
}
