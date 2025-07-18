package common_test

import (
	"testing"

	"github.com/ogero/stremio-subdivx/internal/common"
)

func TestValidateIMDBTitleID(t *testing.T) {
	tests := []struct {
		title   string
		wantErr bool
	}{
		{"tt1234567", false},
		{"tt0012345", false},
		{"tt0", false},
		{"tt", true},
		{"tt-1", true},
		{"tt-1", true},
		{"1234567", true},
		{"ttabcdefg", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			err := common.ValidateIMDBTitleID(tt.title)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIMDBTitleID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSubtitleType(t *testing.T) {
	tests := []struct {
		t       string
		wantErr bool
	}{
		{"movie", false},
		{"series", false},
		{"documentary", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.t, func(t *testing.T) {
			err := common.ValidateSubtitleType(tt.t)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSubtitleType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSubdivxSubtitleID(t *testing.T) {
	tests := []struct {
		id      string
		wantErr bool
	}{
		{"123456", false},
		{"0", true},
		{"-123456", true},
		{"abc123", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			err := common.ValidateSubdivxSubtitleID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSubdivxSubtitleID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
