ver=`git log --pretty=format:"%h" -1`
tag=`git describe --abbrev=0 --tags`
build:
	go build -ldflags "-X main._version_=$(ver) -X main._tag_=$(tag)"
