package python

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
#pip3 --disable-pip-version-check install -r requirements.txt --no-compile --no-warn-script-location
uv pip install -r requirements.txt  --system
#pip install  -r requirements.txt
# coverage run -m pytest && coverage xml
#chmod 0755 -R /root/.cache/pip
`
}

func verboseScript(bs *BuildScript) string {
	return `#!/bin/sh
set -xe
#pip3 --disable-pip-version-check install -r requirements.txt --no-compile --no-warn-script-location
uv pip install -r requirements.txt --system
#pip install -r requirements.txt
# coverage run -m pytest && coverage xml
#chmod 0755 -R /root/.cache/pip
`
}
