package output

import "io"

// Formatter handles output formatting (table or JSON).
type Formatter struct {
	Writer   io.Writer
	JSONMode bool
}
