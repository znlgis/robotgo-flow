//go:build !windows

package serve

import "fmt"

var newServeEngine = func(workDir string) serveEngine {
	panic(fmt.Sprintf("当前平台不支持真实 serve 引擎: %s", workDir))
}
