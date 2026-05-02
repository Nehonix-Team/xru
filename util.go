/***************************************************************************
 * XFPM — XRU package internal utilities
 ***************************************************************************** */

package xru

import "os"

// readFile is the single I/O primitive used by the package.
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
