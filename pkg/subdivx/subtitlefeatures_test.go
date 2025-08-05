package subdivx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubtitleScore(t *testing.T) {
	tests := []struct {
		name          string
		subtitle      Subtitle
		input         string
		expectedScore int
	}{
		{
			name: "empty input",
			subtitle: Subtitle{
				DescriptionWords: []string{"hello", "world"},
			},
			input:         "",
			expectedScore: 0,
		},
		{
			name: "no matches",
			subtitle: Subtitle{
				DescriptionWords: []string{"hello", "world"},
			},
			input:         "test input string",
			expectedScore: 0,
		},
		{
			name: "one match",
			subtitle: Subtitle{
				DescriptionWords: []string{"hello", "world"},
			},
			input:         "hello everyone",
			expectedScore: 1,
		},
		{
			name: "multiple matches",
			subtitle: Subtitle{
				DescriptionWords: []string{"hello", "world", "example"},
			},
			input:         "hello world example test",
			expectedScore: 3,
		},
		{
			name: "case insensitive matching",
			subtitle: Subtitle{
				DescriptionWords: []string{"hello", "world"},
			},
			input:         "HELLO world",
			expectedScore: 2,
		},
		{
			name: "duplicates in input string",
			subtitle: Subtitle{
				DescriptionWords: []string{"hello", "world"},
			},
			input:         "hello hello world world",
			expectedScore: 2,
		},
		{
			name: "trailing and leading spaces",
			subtitle: Subtitle{
				DescriptionWords: []string{"hello", "world"},
			},
			input:         "   hello    world   ",
			expectedScore: 2,
		},
		{
			name: "special characters in input",
			subtitle: Subtitle{
				DescriptionWords: []string{"hello", "world"},
			},
			input:         "hello! @world!!",
			expectedScore: 2,
		},
		{
			name: "special characters and no spaces",
			subtitle: Subtitle{
				DescriptionWords: []string{"hello", "world"},
			},
			input:         "hello!@world!!",
			expectedScore: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualScore := tt.subtitle.Score(tt.input)
			assert.Equal(t, tt.expectedScore, actualScore)
		})
	}
}

func TestAlphaNumericWords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "input with only spaces",
			input:    "     ",
			expected: []string{},
		},
		{
			name:     "special characters only",
			input:    "!@#$%^&*()_+-=[]{}|;:',.<>?/`~",
			expected: []string{},
		},
		{
			name:     "alphanumeric word",
			input:    "hello",
			expected: []string{"hello"},
		},
		{
			name:     "alphanumeric words",
			input:    "hello world 123",
			expected: []string{"hello", "world", "123"},
		},
		{
			name:     "repeated alphanumeric words",
			input:    "hello world 123!world",
			expected: []string{"hello", "world", "123"},
		},
		{
			name:     "mixed case words",
			input:    "HeLLo WoRLd",
			expected: []string{"hello", "world"},
		},
		{
			name:     "input with numbers and spaces",
			input:    "123 456 789abc",
			expected: []string{"123", "456", "789abc"},
		},
		{
			name:     "mixed letters, numbers, and special characters",
			input:    "abc!def123&ghi 456##",
			expected: []string{"abc", "def123", "ghi", "456"},
		},
		{
			name:     "non-alphanumeric with spaces",
			input:    "@@@ *** $$$ abc 123 $$$ @@@",
			expected: []string{"abc", "123"},
		},
		{
			name:     "trailing and leading spaces",
			input:    "   hello world   ",
			expected: []string{"hello", "world"},
		},
		{
			name:     "input with consecutive spaces",
			input:    "hello     world  123",
			expected: []string{"hello", "world", "123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := alphaNumericDistinctLowercaseWords(tt.input)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
