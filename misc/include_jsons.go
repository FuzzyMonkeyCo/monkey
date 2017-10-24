package main

import (
	"io"
	"os"
)

var SCHEMAS = map[string]string{
	"REQv1": "misc/req_v1.json",
	"CMDv1": "misc/cmd_req_v1.json",
}

func main() {
	out, _ := os.Create("schemas.go")
	out.Write([]byte("package main \n\nconst (\n"))
	for name, path := range SCHEMAS {
		out.Write([]byte(name + " = `"))
		fd, err := os.Open(path)
		if err != nil {
			panic(err)
		}
		defer fd.Close()
		io.Copy(out, fd)
		out.Write([]byte("`\n"))
	}
	out.Write([]byte(")\n"))
}
