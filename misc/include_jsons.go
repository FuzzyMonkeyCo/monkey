package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	schemas := map[string]string{
		"schemaREQv1":     "misc/req_v1.json",
		"schemaCMDv1":     "misc/cmd_req_v1.json",
		"schemaCMDDonev1": "misc/cmd_rep_done_v1.json",
	}

	out, err := os.Create("schemas.go")
	if err != nil {
		panic(err)
	}
	defer out.Close()

	if _, err := fmt.Fprintln(out, "package main"); err != nil {
		panic(err)
	}
	fmt.Fprintln(out, "import \"log\"")
	fmt.Fprintln(out, "import \"gopkg.in/xeipuuv/gojsonschema.v0\"")

	var initFunc bytes.Buffer
	fmt.Fprintln(&initFunc, "func loadSchemas() {")

	for name, path := range schemas {
		fmt.Fprintf(out, "var %s gojsonschema.JSONLoader\n", name)
		Name := strings.Title(name)
		fmt.Fprintf(out, "func isValidFor%s(data []byte) (bool, error) {\n", Name)
		fmt.Fprintln(out, "    loader := gojsonschema.NewStringLoader(string(data))")
		fmt.Fprintf(out, "    is, err := gojsonschema.Validate(%s, loader)\n", name)
		fmt.Fprintln(out, "    if err != nil {")
		fmt.Fprintln(out, "        log.Println(\"[ERR]\", err)")
		fmt.Fprintln(out, "        return false, err")
		fmt.Fprintln(out, "    }")
		fmt.Fprintln(out, "    return is.Valid(), nil")
		fmt.Fprintln(out, "}")

		if name != "schemaREQv1" {
			fd, err := os.Open(path)
			if err != nil {
				panic(err)
			}
			defer fd.Close()

			fmt.Fprintf(&initFunc, "\t%s = gojsonschema.NewStringLoader(`", name)
			io.Copy(&initFunc, fd)
			fmt.Fprintln(&initFunc, "`)")
		} else {
			fmt.Fprintf(&initFunc, "\t%s = gojsonschema.NewStringLoader(`", name)
			if err := writeReqSchema(&initFunc); err != nil {
				panic(err)
			}
			fmt.Fprintln(&initFunc, "`)")
		}
	}

	fmt.Fprintln(&initFunc, "}")
	io.Copy(out, &initFunc)
}

func writeReqSchema(fd *bytes.Buffer) (err error) {
	harStr, err := ioutil.ReadFile("misc/har_1.2.json")
	if err != nil {
		return
	}

	var harDefs struct {
		Defs map[string]interface{} `json:"definitions"`
	}
	if err = json.Unmarshal(harStr, &harDefs); err != nil {
		return
	}

	reqStr, err := ioutil.ReadFile("misc/req_v1.json")
	if err != nil {
		return
	}

	var reqJSON map[string]interface{}
	if err = json.Unmarshal(reqStr, &reqJSON); err != nil {
		return
	}

	defs := reqJSON["definitions"].(map[string]interface{})
	for key, def := range harDefs.Defs {
		defs[key] = def
	}

	writer := json.NewEncoder(fd)
	err = writer.Encode(reqJSON)
	return
}
