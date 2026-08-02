.PHONY: build build-ort test test-ort bench-perf bench-perf-ort clean

# libtokenizers.a is required for ORT builds (CGo dependency of daulet/tokenizers).
# Download for darwin/arm64:
#   curl -fSL https://github.com/daulet/tokenizers/releases/download/v1.26.0/libtokenizers.darwin-aarch64.tar.gz \
#       | tar -xz -C ~/lib/
# Then set: export CGO_LDFLAGS="-L${HOME}/lib"
#
# Or install system-wide (requires sudo):
#   sudo cp ~/lib/libtokenizers.a /usr/local/lib/libtokenizers.a

# Default build (pure Go, no CGo required, works everywhere)
build:
	go build ./...

# ORT build — enables ONNX Runtime + CoreML on darwin/arm64.
# Requires CGo and libtokenizers.a. Set CGO_LDFLAGS if library is not in a
# standard search path (e.g. CGO_LDFLAGS="-L${HOME}/lib").
# The bundled arm64 dylib inside yalue/onnxruntime_go@v1.27.0 already links
# CoreML.framework and Metal.framework — no separate ORT download needed.
build-ort:
	go build -tags ORT ./...

# Run all tests (pure Go backend)
test:
	go test ./...

# Run all tests with ORT backend (darwin/arm64: CoreML accelerated).
# Requires CGo and libtokenizers.a.
test-ort:
	go test -tags ORT ./...

# Run the performance benchmark (pure Go)
bench-perf:
	go run ./cmd/benchperf/

# Run the performance benchmark with ORT + CoreML.
# On darwin/arm64 this uses the CoreML EP (ANE/GPU) where supported.
bench-perf-ort:
	go run -tags ORT ./cmd/benchperf/

clean:
	rm -f mempalace mempalace-go mempalace-test coverage.out
