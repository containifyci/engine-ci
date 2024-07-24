package maven

type Image string

type BuildScript struct {
	Verbose bool
}

func NewBuildScript(verbose bool) *BuildScript {
	return &BuildScript{
		Verbose: verbose,
	}
}

func Script(bs *BuildScript) string {
	if bs.Verbose {
		return verboseScript(bs)
	}
	return simpleScript(bs)
}

func simpleScript(bs *BuildScript) string {
	return `#!/bin/sh
set -xe
./mvnw --batch-mode package
`
}

func verboseScript(bs *BuildScript) string {
	return `#!/bin/sh
set -xe
./mvnw --batch-mode package -X
`
}
