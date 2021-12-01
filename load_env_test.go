package raftkv

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadString(t *testing.T) {
	var value string

	tests := []struct {
		name      string
		key       string
		setEnv    func(t *testing.T)
		wantValue string
		wantErr   error
	}{
		{
			name:      "key is missing",
			key:       "SOME_KEY",
			wantValue: "",
			wantErr:   fmt.Errorf("SOME_KEY is missing"),
		},
		{
			name: "get a value",
			key:  "SOME_KEY",
			setEnv: func(t *testing.T) {
				t.Setenv("SOME_KEY", "some_value")
			},
			wantValue: "some_value",
			wantErr:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv != nil {
				tt.setEnv(t)
			}
			err := LoadString(tt.key, &value)
			if tt.wantErr == nil {
				assert.Nil(t, err)
			} else {
				assert.Equal(t, tt.wantErr, err)
			}
			assert.Equal(t, tt.wantValue, value)
		})
	}
}

func TestLoadAddr(t *testing.T) {
	var value string

	tests := []struct {
		name      string
		key       string
		setEnv    func(t *testing.T)
		wantValue string
		wantErr   error
	}{
		{
			name:      "key is missing",
			key:       "SOME_ADDR",
			wantValue: "",
			wantErr:   fmt.Errorf("SOME_ADDR is missing"),
		},
		{
			name: "get an address",
			key:  "SOME_ADDR",
			setEnv: func(t *testing.T) {
				t.Setenv("SOME_ADDR", "localhost:9000")
			},
			wantValue: "localhost:9000",
			wantErr:   nil,
		},
		{
			name: "get an omitted address",
			key:  "SOME_ADDR",
			setEnv: func(t *testing.T) {
				t.Setenv("SOME_ADDR", ":9000")
			},
			wantValue: ":9000",
			wantErr:   nil,
		},
		{
			name: "get an omitted address",
			key:  "SOME_ADDR",
			setEnv: func(t *testing.T) {
				t.Setenv("SOME_ADDR", "9000")
			},
			wantValue: ":9000",
			wantErr:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv != nil {
				tt.setEnv(t)
			}
			err := LoadAddr(tt.key, &value)
			if tt.wantErr == nil {
				assert.Nil(t, err)
			} else {
				assert.Equal(t, tt.wantErr, err)
			}
			assert.Equal(t, tt.wantValue, value)
		})
	}
}

func TestLoadInt(t *testing.T) {
	var value int

	tests := []struct {
		name      string
		key       string
		setEnv    func(t *testing.T)
		wantValue int
		wantErr   error
	}{
		{
			name:      "key is missing",
			key:       "SOME_KEY",
			wantValue: 0,
			wantErr:   fmt.Errorf("SOME_KEY is missing"),
		},
		{
			name: "get an interger",
			key:  "SOME_KEY",
			setEnv: func(t *testing.T) {
				t.Setenv("SOME_KEY", "100")
			},
			wantValue: 100,
			wantErr:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv != nil {
				tt.setEnv(t)
			}
			err := LoadInt(tt.key, &value)
			if tt.wantErr == nil {
				assert.Nil(t, err)
			} else {
				assert.Equal(t, tt.wantErr, err)
			}
			assert.Equal(t, tt.wantValue, value)
		})
	}
}
