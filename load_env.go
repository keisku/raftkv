package raftkv

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

func LoadString(key string, value *string) error {
	v := os.Getenv(key)
	if v == "" {
		return fmt.Errorf("%s is missing", key)
	}
	*value = v
	return nil
}

func LoadAddr(key string, value *string) error {
	if err := LoadString(key, value); err != nil {
		return err
	}
	if !strings.HasPrefix(*value, ":") && unicode.IsDigit(rune((*value)[0])) {
		*value = fmt.Sprintf(":%s", *value)
	}
	return nil
}

func LoadInt(key string, value *int) error {
	v := os.Getenv(key)
	if v == "" {
		return fmt.Errorf("%s is missing", key)
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("%s is an invalid value: %w", key, err)
	}
	*value = i
	return nil
}
