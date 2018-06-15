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
	fmt.Fprintln(out, "import \"github.com/xeipuuv/gojsonschema\"")

	var initFunc bytes.Buffer
	fmt.Fprintln(&initFunc, "func loadSchemas() {")
	fmt.Fprintln(&initFunc, "\tvar err error")

	for name, path := range schemas {
		fmt.Fprintf(out, "var %s *gojsonschema.Schema\n", name)
		Name := strings.Title(name)
		fmt.Fprintf(out, "func isValidFor%s(data []byte) (bool, error) {\n", Name)
		fmt.Fprintln(out, "\tloader := gojsonschema.NewStringLoader(string(data))")
		fmt.Fprintf(out, "\tis, err := %s.Validate(loader)\n", name)
		fmt.Fprintln(out, "\tif err != nil {")
		fmt.Fprintln(out, "\t\tlog.Println(\"[ERR]\", err)")
		fmt.Fprintln(out, "\t\treturn false, err")
		fmt.Fprintln(out, "\t}")
		fmt.Fprintln(out, "\treturn is.Valid(), nil")
		fmt.Fprintln(out, "}")

		loader := name + "Loader"
		if name != "schemaREQv1" {
			fd, err := os.Open(path)
			if err != nil {
				panic(err)
			}
			defer fd.Close()

			fmt.Fprintf(&initFunc, "\t%s := gojsonschema.NewStringLoader(`", loader)
			if _, err := io.Copy(&initFunc, fd); err != nil {
				panic(err)
			}
			fmt.Fprintln(&initFunc, "`)")
			fmt.Fprintf(&initFunc, "\tif %s, err = gojsonschema.NewSchema(%s); err != nil { panic(err) }\n", name, loader)
		} else {
			fmt.Fprintf(&initFunc, "\t%s := gojsonschema.NewStringLoader(`", loader)
			if err := writeReqSchema(&initFunc); err != nil {
				panic(err)
			}
			fmt.Fprintln(&initFunc, "`)")
			fmt.Fprintf(&initFunc, "\tif %s, err = gojsonschema.NewSchema(%s); err != nil { panic(err) }\n", name, loader)
		}
	}

	fmt.Fprintln(&initFunc, "}")
	if _, err := io.Copy(out, &initFunc); err != nil {
		panic(err)
	}
}

func writeReqSchema(fd io.Writer) (err error) {
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

	err = json.NewEncoder(fd).Encode(reqJSON)
	return
}
