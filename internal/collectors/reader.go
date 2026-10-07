// reader.go provides the testability seam for collectors.
//
// Production code reads the real filesystem; tests override readFile
// with fixture data. One variable, no framework.
package collectors

import "os"

// readFile reads a sysfs/procfs/config file. Overridable in tests.
var readFile = os.ReadFile
