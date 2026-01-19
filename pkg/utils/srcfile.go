package utils

import "path/filepath"

type SrcFile string

func NewSrcFile(folder, file string) SrcFile {
	return SrcFile(filepath.Join(folder, file))
}

func (s SrcFile) IsEmpty() bool {
	return s == ""
}

func (s SrcFile) IsNotEmpty() bool {
	return !s.IsEmpty()
}

func (s SrcFile) Container() string {
	if s != "/src/main.go" && s.IsNotEmpty() {
		return "/src/" + string(s)
	}
	return string(s)
}

func (s SrcFile) Host() string {
	return string(s)
}
