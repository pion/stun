// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

//go:build !js
// +build !js

package stun

import (
	"bytes"
	"encoding/csv"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func loadCSV(tb testing.TB, name string) [][]string {
	tb.Helper()

	data := loadData(tb, name)
	r := csv.NewReader(bytes.NewReader(data))
	r.Comment = '#'
	records, err := r.ReadAll()
	assert.NoError(tb, err)

	return records
}

func TestIANA(t *testing.T) { //nolint:cyclop
	t.Run("Methods", func(t *testing.T) {
		records := loadCSV(t, "stun-parameters-2.csv")
		methods := make(map[string]Method)
		for _, r := range records[1:] {
			var (
				v    = r[0]
				name = r[1]
			)
			if strings.Contains(v, "-") {
				continue
			}
			val, parseErr := strconv.ParseInt(v[2:], 16, 64)
			assert.NoError(t, parseErr)
			t.Logf("value: 0x%x, name: %s", val, name)
			methods[name] = Method(val) //nolint:gosec // G115
		}
		for val, name := range methodName() {
			mapped, ok := methods[name]
			assert.True(t, ok, "failed to find method %s in IANA", name)
			assert.Equal(t, mapped, val, "%s: IANA %d != actual %d", name, mapped, val)
		}
	})
	t.Run("Attributes", func(t *testing.T) {
		records := loadCSV(t, "stun-parameters-4.csv")
		attrTypes := map[string]AttrType{}
		for _, r := range records[1:] {
			var (
				v    = r[0]
				name = r[1]
			)
			if strings.Contains(v, "-") {
				continue
			}
			val, parseErr := strconv.ParseInt(v[2:], 16, 64)
			assert.NoError(t, parseErr)
			t.Logf("value: 0x%x, name: %s", val, name)
			attrTypes[name] = AttrType(val) //nolint:gosec // G115
		}
		// Not registered in IANA.
		for k, v := range map[string]AttrType{
			"ORIGIN": 0x802F,
		} {
			attrTypes[k] = v
		}
		for val, name := range attrNames() {
			mapped, ok := attrTypes[name]
			assert.True(t, ok, "failed to find attribute %s in IANA", name)
			assert.Equal(t, mapped, val, "%s: IANA %d != actual %d", name, mapped, val)
		}
	})
	t.Run("ErrorCodes", func(t *testing.T) {
		records := loadCSV(t, "stun-parameters-6.csv")
		errorCodes := map[string]ErrorCode{}
		for _, r := range records[1:] {
			var (
				v    = r[0]
				name = r[1]
			)
			if strings.Contains(v, "-") {
				continue
			}
			val, parseErr := strconv.ParseInt(v, 10, 64)
			assert.NoError(t, parseErr)
			t.Logf("value: 0x%x, name: %s", val, name)
			errorCodes[name] = ErrorCode(val)
		}
		for val, nameB := range errorReasons {
			name := string(nameB)
			mapped, ok := errorCodes[name]
			assert.True(t, ok, "failed to find error code %s in IANA", name)
			assert.Equal(t, mapped, val, "%s: IANA %d != actual %d", name, mapped, val)
		}
	})
}
