// Command cryptocli encrypts/decrypts values using the same AES-256-GCM scheme
// as the app, for local debugging of stored "enc:" values.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/abhinavxd/libredesk/internal/crypto"
)

func main() {
	var (
		key     = flag.String("key", "", "32-char encryption key")
		encrypt = flag.Bool("e", false, "encrypt instead of decrypt")
	)
	flag.Parse()

	val := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if val == "" || *key == "" {
		fmt.Fprintln(os.Stderr, "usage: cryptocli [-e] -key KEY <value>")
		os.Exit(2)
	}

	var (
		out string
		err error
	)
	if *encrypt {
		out, err = crypto.Encrypt(val, *key)
	} else {
		out, err = crypto.Decrypt(val, *key)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println(out)
}
