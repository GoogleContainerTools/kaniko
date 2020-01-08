package copy

import (
	"fmt"
	"os"
)

func ExampleCopy() {

	err := Copy("testdata/example", "testdata.copy/example")
	fmt.Println("Error:", err)
	info, _ := os.Stat("testdata.copy/example")
	fmt.Println("IsDir:", info.IsDir())

	// Output:
	// Error: <nil>
	// IsDir: true
}
