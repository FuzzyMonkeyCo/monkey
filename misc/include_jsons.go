package main

import (
	"io"
	"os"
)

func main() {
	schemas := map[string]string{
		"schemaREQv1": "misc/req_v1.json",
		"schemaCMDv1": "misc/cmd_req_v1.json",
		"schemaCMDDonev1": "misc/cmd_rep_done_v1.json",
	}

	out, _ := os.Create("schemas.go")
	out.Write([]byte("package main \n\nconst (\n"))
	for name, path := range schemas {
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
