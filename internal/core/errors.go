package core

import "errors"

// ErrSourceExhausted is returned by a Source when it has no more records to emit.
var ErrSourceExhausted = errors.New("source exhausted")

// ErrSkipped is returned by a Transform to indicate the record should be dropped.
// It is not treated as a pipeline error.
var ErrSkipped = errors.New("record skipped")

// ErrSinkClosed is returned when Write is called on a closed Sink.
var ErrSinkClosed = errors.New("sink is closed")
