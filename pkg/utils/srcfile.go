package utils

type SrcFile string

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
