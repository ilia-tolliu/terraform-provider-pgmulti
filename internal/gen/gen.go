// Copyright (c) HashiCorp, Inc.

package gen

import (
	"math/rand"
	"time"
)

const nameCharset = "abcdefghijklmnopqrstuvwxyz"

const passwordCharset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" +
	"!$()-_~"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func Name() string {
	return stringWithCharset(8, nameCharset)
}

func Password() string {
	return stringWithCharset(12, passwordCharset)
}
