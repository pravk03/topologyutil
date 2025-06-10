// SPDX-FileClonerightText: Cloneright (C) SchedMD LLC.
// SPDX-License-Identifier: Apache-2.0

package bitmaputil

import (
	"reflect"
	"testing"

	"github.com/kelindar/bitmap"
)

func newFrom(s string) bitmap.Bitmap {
	out, err := NewFrom(s)
	if err != nil {
		panic("failed to do NewFrom()")
	}
	return out
}

func TestNew(t *testing.T) {
	type args struct {
		idx []int
	}
	tests := []struct {
		name string
		args args
		want bitmap.Bitmap
	}{
		{
			name: "empty",
			args: args{
				idx: []int{},
			},
			want: bitmap.Bitmap{},
		},
		{
			name: "one",
			args: args{
				idx: []int{0},
			},
			want: bitmap.Bitmap{0x1},
		},
		{
			name: "blocks of data",
			args: args{
				idx: []int{0, 65},
			},
			want: bitmap.Bitmap{0x1, 0x2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.idx...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewFrom(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    bitmap.Bitmap
		wantErr bool
	}{
		{
			name: "empty",
			args: args{
				s: "0x",
			},
			want:    New(),
			wantErr: false,
		},
		{
			name: "data",
			args: args{
				s: "0x1",
			},
			want:    New(0),
			wantErr: false,
		},
		{
			name: "bad data",
			args: args{
				s: "bad data",
			},
			want:    New(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewFrom(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFrom() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewFrom() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestString(t *testing.T) {
	type args struct {
		bm bitmap.Bitmap
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "[New] empty",
			args: args{
				bm: New(),
			},
			want: "0x",
		},
		{
			name: "[New] one",
			args: args{
				bm: New(0),
			},
			want: "0x1",
		},
		{
			name: "[New] blocks of data",
			args: args{
				bm: New(0, 65),
			},
			want: "0x20000000000000001",
		},
		{
			name: "[NewFrom] empty",
			args: args{
				bm: newFrom("0x"),
			},
			want: "0x",
		},
		{
			name: "[NewFrom] one",
			args: args{
				bm: newFrom("0x1"),
			},
			want: "0x1",
		},
		{
			name: "[NewFrom] blocks of data",
			args: args{
				bm: newFrom("0x20000000000000001"),
			},
			want: "0x20000000000000001",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := String(tt.args.bm); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
