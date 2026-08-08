package output

// CompressMode selects how results.json is written to disk.
type CompressMode string

const (
	// CompressNone writes compact (unindented) results.json.
	CompressNone CompressMode = "none"
	// CompressZstd writes results.json.zst (canonical for sweeps).
	CompressZstd CompressMode = "zstd"
)

// ExportOptions controls report artifact layout and compression.
type ExportOptions struct {
	// Compress defaults to CompressZstd when empty.
	Compress CompressMode
	// OfflineDataJS writes data.js (window.__REPORT__) plus a thin report.html
	// that loads it — for file:// open without a local server. Prefer false for
	// sweeps so disk stays near the zstd size.
	OfflineDataJS bool
	// WriteViewer writes viewer.html into the run/sweep directory. When false,
	// the caller (e.g. sweep) is expected to write a shared viewer once at the root.
	WriteViewer bool
}

// DefaultExportOptions returns zstd compression with no offline data.js.
func DefaultExportOptions() ExportOptions {
	return ExportOptions{
		Compress:      CompressZstd,
		OfflineDataJS: false,
		WriteViewer:   false,
	}
}

func (o ExportOptions) withDefaults() ExportOptions {
	if o.Compress == "" {
		o.Compress = CompressZstd
	}
	return o
}
