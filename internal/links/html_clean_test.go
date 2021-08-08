// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/6/27

package links

import (
	"reflect"
	"testing"
)

func Test_clean(t *testing.T) {
	type args struct {
		code []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "br 1",
			args: args{
				code: []byte("abc <br/><br/> hello"),
			},
			want: []byte("abc <br> hello"),
		},
		{
			name: "br 2",
			args: args{
				code: []byte("abc <br><br/> hello"),
			},
			want: []byte("abc <br> hello"),
		},
		{
			name: "br 3",
			args: args{
				code: []byte("abc <br>\n<br/> hello"),
			},
			want: []byte("abc <br> hello"),
		},
		{
			name: "br 4",
			args: args{
				code: []byte("abc <br>\n<br/>\r\n hello"),
			},
			want: []byte("abc <br>\nhello"),
		},
		{
			name: "br 5",
			args: args{
				code: []byte("abc < BR>\n<bR />\r\n hello"),
			},
			want: []byte("abc <br>\nhello"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clean(tt.args.code); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("clean() = %q, want %q", got, tt.want)
			}
		})
	}
}
