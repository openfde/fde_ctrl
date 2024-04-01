

all: build

ver=`git log --pretty=format:"%h" -1`
tag=`git describe --abbrev=0 --tags`
date1=`date +%F_%T`

build:
	go build -o fde_brightness  -ldflags "-X main._version_=$(ver) -X main._tag_=$(tag) -X main._date_=$(date1)"


