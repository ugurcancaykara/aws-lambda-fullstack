build::
	cd processinglambda && GOOS=linux GOARCH=amd64 go build -o ./bootstrap ./main.go
	zip -j ./processinglambda/deployment.zip ./processinglambda/bootstrap
