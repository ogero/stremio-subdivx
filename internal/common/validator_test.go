package common_test

import (
	"testing"

	"github.com/ogero/stremio-subdivx/internal/common"
	"github.com/stretchr/testify/assert"
)

func TestValidateIMDBTitleID(t *testing.T) {
	tests := []struct {
		title   string
		wantErr assert.ErrorAssertionFunc
	}{
		{"tt1234567", assert.NoError},
		{"tt0012345", assert.NoError},
		{"tt0", assert.NoError},
		{"tt", assert.Error},
		{"tt-1", assert.Error},
		{"tt-1", assert.Error},
		{"1234567", assert.Error},
		{"ttabcdefg", assert.Error},
		{"", assert.Error},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			err := common.ValidateIMDBTitleID(tt.title)
			tt.wantErr(t, err)
		})
	}
}

func TestValidateSubtitleType(t *testing.T) {
	tests := []struct {
		t       string
		wantErr assert.ErrorAssertionFunc
	}{
		{"movie", assert.NoError},
		{"series", assert.NoError},
		{"documentary", assert.Error},
		{"", assert.Error},
	}

	for _, tt := range tests {
		t.Run(tt.t, func(t *testing.T) {
			err := common.ValidateSubtitleType(tt.t)
			tt.wantErr(t, err)
		})
	}
}

func TestValidateSubdivxSubtitleID(t *testing.T) {
	tests := []struct {
		id      string
		wantErr assert.ErrorAssertionFunc
	}{
		{"123456", assert.NoError},
		{"0", assert.Error},
		{"-123456", assert.Error},
		{"abc123", assert.Error},
		{"", assert.Error},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			err := common.ValidateSubdivxSubtitleID(tt.id)
			tt.wantErr(t, err)
		})
	}
}
