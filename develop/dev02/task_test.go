package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnpack(t *testing.T) {
	tests := []struct {
		arg     string
		want    string
		wantErr bool
	}{
		{
			arg:     "a4bc2d5e",
			want:    "aaaabccddddde",
			wantErr: false,
		},
		{
			arg:     "世9Ж5О0",
			want:    "世世世世世世世世世ЖЖЖЖЖ",
			wantErr: false,
		},
		{
			arg:     "abcd",
			want:    "abcd",
			wantErr: false,
		},
		{
			arg:     "45",
			want:    "",
			wantErr: true,
		},
		{
			arg:     "",
			want:    "",
			wantErr: false,
		},
		{
			arg:     `qwe\4\5`,
			want:    "qwe45",
			wantErr: false,
		},
		{
			arg:     `qwe\45`,
			want:    "qwe44444",
			wantErr: false,
		},
		{
			arg:     `qwe\\5`,
			want:    `qwe\\\\\`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			got, err := Unpack(tt.arg)
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrIncorrectString)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
