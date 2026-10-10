# Native embedding lifetime

Actual App recall under race revealed session destruction concurrent with GoMLX execution. The pinned Hugot backend returns immediately when the request context expires, leaving a native errgroup worker running. A direct real ONNX deadline/Close probe reproduces DATA RACE.

Keep the pinned dependency unchanged. Serialize embedding execution and session destruction with an owner mutex. Check cancellation before native admission and after completion; pass a context retaining values but without cancellation into the native pipeline so Hugot joins its workers before releasing the mutex. ContextHost's own deadline and fail-closed projection remain unchanged. Close is idempotent; calls after close return ErrClosed. Apply the same join rule to batch and fallback inference.

Tradeoff: in-progress native inference cannot be interrupted by this backend; releasing model resources waits for it to finish. No deadline extension in ContextHost and no positive recall fabrication. A native computation hang still requires process supervision; this increment does not claim a hard shutdown latency bound.
