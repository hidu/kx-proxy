// Copyright(C) 2021 github.com/fsgo  All Rights Reserved.
// Author: fsgo
// Date: 2021/6/27

package util

import (
	"reflect"
	"testing"
)

func TestExtensions_Rewrite(t *testing.T) {
	type args struct {
		body []byte
	}
	tests := []struct {
		name string
		es   Extensions
		args args
		want []byte
	}{
		{
			name: "no image input 1",
			es:   Extensions{"no_images"},
			args: args{
				body: []byte(`<input class="abc" type="image" src="GjCDco9qvdV" style="">`),
			},
			want: []byte(`<input class="abc" type="button" img_ignore_src="GjCDco9qvdV" style="">`),
		},
		{
			name: "no image input 2",
			es:   Extensions{"no_images"},
			args: args{
				body: []byte(`<input type=image src="GjCDco9qvdV" style="">`),
			},
			want: []byte(`<input type=button img_ignore_src="GjCDco9qvdV" style="">`),
		},
		{
			name: "no image img 1",
			es:   Extensions{"no_images"},
			args: args{
				body: []byte(`<img src="" abcd>`),
			},
			want: []byte(`<!-- img ignore -->`),
		},
		{
			name: "no image img 2",
			es:   Extensions{"no_images"},
			args: args{
				body: []byte(`<Img style="" src="" abcd class="abcd">`),
			},
			want: []byte(`<!-- img ignore -->`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.es.Rewrite(tt.args.body); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Rewrite() = %s,\n want %s", got, tt.want)
			}
		})
	}
}
