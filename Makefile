

all: git2Control

git2Control: git2Control.go 
	GOOS=linux GOARCH=arm GOARM=6 go build  -o binaries/linuxArm/git2Control git2Control.go
	GOOS=linux GOARCH=386 GOARM=6 go build  -o binaries/linux32/git2Control git2Control.go
	GOOS=linux GOARCH=amd64 GOARM=6 go build  -o binaries/linux64/git2Control git2Control.go
	GOOS=windows GOARCH=amd64 GOARM=6 go build  -o binaries/windows64/git2Control git2Control.go
	GOOS=windows GOARCH=amd64 GOARM=6 go build  -o binaries/windows32/git2Control git2Control.go
	GOOS=darwin GOARCH=amd64 GOARM=6 go build  -o binaries/mac/git2Control git2Control.go

