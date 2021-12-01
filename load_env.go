package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

func loadString(key string, value *string) error {
	v := os.Getenv(key)
	if v == "" {
		return fmt.Errorf("%s is missing", key)
	}
	*value = v
	return nil
}

func loadAddr(key string, value *string) error {
	if err := loadString(key, value); err != nil {
		return err
	}
	if !strings.HasPrefix(*value, ":") && unicode.IsDigit(rune((*value)[0])) {
		*value = fmt.Sprintf(":%s", *value)
	}
	return nil
}

func loadInt(key string, value *int) error {
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
