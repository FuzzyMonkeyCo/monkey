package main

import (
	"fmt"
	"time"
	"os"
	"math/rand"
	. "github.com/logrusorgru/aurora"
)

// 0 monkey master $ ./monkey fuzz
// No validation errors found.
// ✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✓✗
// ✗✗✗✓✓✓✓✓✓
// Ran 37 tests totalling 314 requests
// A bug was detected after 48 tests then shrunk 7 times!
// 6 monkey master $

var r *rand.Rand
var p Value
func init() {
	r = rand.New(rand.NewSource(99))
	p = Blue("")
}

func yellow(s string) string { return "\033[0;33m" + s + "\033[0m" }

func spin(isOK bool) {
	i := 0
	sp := []string{"◐", "◓", "◑", "◒"}
	//sp := []string{"◡", "◡", " ", "⊙", "⊙", " ", "◠", "◠"}
	// sp := []string{"▉", "▊", "▋", "▌", "▍", "▎", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}
	//sp := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	K := r.Intn(30)
	for k := 0; k < K; k++ {
		i += 1
		fmt.Printf("\b%s", Blue(sp[i%len(sp)]))
		time.Sleep(40 *  time.Millisecond)
	}

	ok, ko := Green("✓"), Red("✗")
	if i == 0 {
		if isOK {
			fmt.Printf("%s", ok)
		} else {
			fmt.Printf("%s", ko)
		}
	} else {
		if isOK {
			fmt.Printf("\b%s%s", p, ok)
		} else {
			fmt.Printf("\b%s%s", p, ko)
		}
	}
	if isOK { p = ok} else { p = ko }
}

func main() {
	time.Sleep(10 * time.Millisecond)
	fmt.Println(yellow("Docs are valid."))
	i := 0
	for {
		// break///////////

		i += 1
		spin(true)
		if i == 27 {
			spin(false)
			fmt.Printf("\n")
			break
		}
	}

	fmt.Println(yellow("Ran 27 tests totalling 132 HTTP requests"))
	i = 0
	for {
		i += 1
		if i < 3 {
			spin(false)
			continue
		}
		if i < 5 {
			spin(true)
			continue
		}
		if i == 5 {
			spin(true)
			fmt.Printf("\n")
			break
		}
	}

	time.Sleep(10 * time.Millisecond)
	fmt.Println(Red("A bug was detected after 27 tests then shrank 4 times!"))
	fmt.Println(yellow("It was minimized down to the following 2 HTTP requests:"))
	fmt.Println("")
	in, out := yellow(">"), yellow("<")

	fmt.Printf("" +
		"# %s\n" +
		"%s %s HTTP/1.1\n" +
		"  %s\n" +
		"%s Host: localhost\n" +
		"%s Connection: keep-alive\n" +
		"%s User-Agent: monkey/0.19.2\n" +
		"%s Accept: application/json\n" +
		"%s\n" +
		"%s HTTP/1.1 201 Created\n" +
		"  %s\n" +
		"%s X-Powered-By: Express\n" +
		"%s Content-Type: application/json\n" +
		"%s Content-Length: 64\n" +
		"%s ETag: W/\"a-bAsFyilMr4Ra1hIU5PyoyFRunpI\"\n" +
		"%s Date: Wed, 21 Mar 2018 11:52:08 GMT\n" +
		"%s\n" +
		"%s {\"data\":{\"category\":\"rabbit\",\"name\":\"Roger\"},\"status\":\"success\"}\n" +
		"%s\n\n",
		Blue("PUT /pets/:category/:name"), in, Bold("PUT /pets/rabbit/Roger"),
		Green("          ~~~~~~ ~~~~"), in, in, in, in, in,
		out, Green("         ~~~~~~~~~~~"), out, out, out, out, out, out, out,
		yellow("in 42ms"))

	fmt.Printf("" +
		"# %s\n" +
		"%s %s HTTP/1.1\n" +
		"  %s\n" +
		"%s Host: localhost\n" +
		"%s Connection: keep-alive\n" +
		"%s User-Agent: monkey/0.19.2\n" +
		"%s Accept: application/json\n" +
		"%s\n" +
		"%s HTTP/1.1 404 Not Found\n" +
		"  %s\n" +
		"%s X-Powered-By: Express\n" +
		"%s Date: Wed, 21 Mar 2018 11:52:09 GMT\n" +
		"%s Connection: close\n" +
		"%s\n" +
		"%s\n\n",
		Blue("GET /pets/:category/:name"), in, Bold("GET /pets/rabbit/Roger"),
		Green("          ~~~~~~ ~~~~"), in, in, in, in, in,
		out, Green("         ^~~~~~~~~~~~~"), out, out, out, out, yellow("in 8ms"))

	fmt.Printf("%s: %s\n\n", Bold(Red("error")), Bold("expected 200 'OK', not 404 'Not Found'"))
	fmt.Println(yellow("This bug has been saved so you can replay it later"))

	os.Exit(6)
}
