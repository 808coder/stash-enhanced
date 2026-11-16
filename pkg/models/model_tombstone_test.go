package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormaliseTombstoneStem(t *testing.T) {
	tests := []struct {
		stem string
		want string
	}{
		{"Studio.24.01.01.Name.XXX.1080p", "Studio.24.01.01.Name.XXX"},
		{"Studio.24.01.01.Name.XXX.720P", "Studio.24.01.01.Name.XXX"},
		{"Studio.24.01.01.Name.XXX.2160p", "Studio.24.01.01.Name.XXX"},
		{"Studio.24.01.01.Name.XXX.4K", "Studio.24.01.01.Name.XXX"},
		{"Studio.24.01.01.Name.XXX", "Studio.24.01.01.Name.XXX"},
		// only a trailing token is stripped
		{"Studio.24.01.01.Name.XXX.1080p.HEVC", "Studio.24.01.01.Name.XXX.1080p.HEVC"},
		// not a resolution token
		{"Clip.10p", "Clip.10p"},
		{"Clip.12345p", "Clip.12345p"},
		{"Clip1080p", "Clip1080p"},
		{"Studio.24.01.01", "Studio.24.01.01"},
		// a stem that is the token only stays as it is
		{".1080p", ".1080p"},
		{"", ""},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, NormaliseTombstoneStem(tt.stem), tt.stem)
	}
}
