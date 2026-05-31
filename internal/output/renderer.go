// internal/output/renderer.go
package output

import "github.com/Viswesh-G/scope/internal/metrics"

type Renderer interface {
	Render(metrics.Report)
}
