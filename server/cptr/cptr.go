// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENCE file for details.

package cptr

func String(v string) *string {
	return &v
}

func Int(v int) *int {
	return &v
}

func Int32(v int32) *int32 {
	return &v
}

func Int64(v int64) *int64 {
	return &v
}

func Bool(v bool) *bool {
	return &v
}
