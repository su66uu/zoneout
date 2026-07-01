package playback

import (
	"io"

	"zoneout/internal/audioanalysis"
)

type AnalyzerReader struct {
	src      io.Reader
	analyzer *audioanalysis.Analyzer
}

func newAnalyzerReader(src io.Reader, analyzer *audioanalysis.Analyzer) io.Reader {
	if analyzer == nil {
		return src
	}

	return &AnalyzerReader{
		src:      src,
		analyzer: analyzer,
	}
}

// This will called by oto internally
func (r *AnalyzerReader) Read(p []byte) (int, error) {
	n, err := r.src.Read(p)
	if n > 0 {
		r.analyzer.ObservePCM16StereoLE(p[:n])
	}

	return n, err
}
