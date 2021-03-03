package utils

import (
	"os"
)

const (
	UserWritable = 1 << (uint(7))
	UserExecutable = 1 << (uint(6))
)

func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

func IsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func IsWritable(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if info.Mode() & UserWritable == 0 {
		return false, nil
	}
	return true, nil
}

func IsExecutable(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if info.Mode() & UserExecutable == 0 {
		return false, nil
	}
	return true, nil
}
