// Copyright(C) 2020 github.com/hidu  All Rights Reserved.
// Author: hidu (duv123+git@baidu.com)
// Date: 2020/12/27

package links

import (
	"reflect"
	"testing"
)

func TestAllLinks(t *testing.T) {
	type args struct {
		body     []byte
		baseHref string
		urlNow   string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "case 1",
			args: args{
				body: []byte(`a<a href="/a"> <a href="/b">`),
			},
			want: nil,
		},
		{
			name: "case 2",
			args: args{
				body:   []byte(`a<a href="/a"> <a href="/b">`),
				urlNow: "http://abc.com/d",
			},
			want: []string{
				"http://abc.com/a",
				"http://abc.com/b",
			},
		},
		{
			name: "case 3",
			args: args{
				body:   []byte(`a<a href="c"> <a href="d">`),
				urlNow: "http://abc.com/a/d",
			},
			want: []string{
				"http://abc.com/a/c",
				"http://abc.com/a/d",
			},
		},
		{
			name: "case 4",
			args: args{
				body:   []byte(`a<a href="http://z.com/a"> <a href="http://z.com/b">`),
				urlNow: "http://abc.com/a/d",
			},
			want: []string{
				"http://z.com/a",
				"http://z.com/b",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AllLinks(tt.args.body, tt.args.baseHref, tt.args.urlNow); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AllLinks() = %v, want %v", got, tt.want)
			}
		})
	}
}
