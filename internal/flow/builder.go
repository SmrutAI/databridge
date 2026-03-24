package flow

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/tushar2708/conveyor"

	"github.com/SmrutAI/databridge/internal/core"
)

// Flow is a configured pipeline ready to run.
// Build one with NewFlow, chain Source/Transform/Sink calls, then call Run.
type Flow struct {
	name       string
	source     core.Source
	transforms []core.Transform
	sinks      []core.Sink
}

// NewFlow creates a new named pipeline builder.
func NewFlow(name string) *Flow {
	return &Flow{name: name}
}

// Source sets the data source for this flow.
func (f *Flow) Source(s core.Source) *Flow {
	f.source = s
	return f
}

// Transform appends a transform stage to the pipeline.
// Transforms are applied in the order they are added.
func (f *Flow) Transform(t core.Transform) *Flow {
	f.transforms = append(f.transforms, t)
	return f
}

// Sink appends a sink to the pipeline.
// All sinks receive every record (fan-out via multiSinkAdapter).
func (f *Flow) Sink(s core.Sink) *Flow {
	f.sinks = append(f.sinks, s)
	return f
}

// Run executes the full pipeline and returns stats when it completes.
// It opens all sources and sinks before starting, closes them on return.
func (f *Flow) Run(ctx context.Context) (*core.FlowStats, error) {
	if f.source == nil {
		return nil, fmt.Errorf("flow %s: no source configured", f.name)
	}
	if len(f.sinks) == 0 {
		return nil, fmt.Errorf("flow %s: no sinks configured", f.name)
	}

	if err := f.source.Open(ctx); err != nil {
		return nil, fmt.Errorf("flow %s: open source: %w", f.name, err)
	}
	defer f.source.Close() //nolint:errcheck

	for i := range f.sinks {
		if err := f.sinks[i].Open(ctx); err != nil {
			return nil, fmt.Errorf("flow %s: open sink %s: %w", f.name, f.sinks[i].Name(), err)
		}
	}
	defer func() {
		for i := range f.sinks {
			_ = f.sinks[i].Close()
		}
	}()

	start := time.Now()
	var recordsIn, recordsOut, recordsSkipped atomic.Int64

	cnv, err := conveyor.NewConveyor(f.name, 100)
	if err != nil {
		return nil, fmt.Errorf("flow %s: create conveyor: %w", f.name, err)
	}

	// Source adapter: streams all records from the source into the conveyor.
	// WorkerModeLoop is correct here — the source owns the channel iteration.
	src := &sourceAdapter{
		ConcreteSourceExecutor: &conveyor.ConcreteSourceExecutor[*core.Record]{
			Name: f.source.Name(),
		},
		source:  f.source,
		counter: &recordsIn,
	}
	if err := conveyor.AddSource[*core.Record](cnv, src, conveyor.WorkerModeLoop); err != nil {
		return nil, fmt.Errorf("flow %s: add source: %w", f.name, err)
	}

	// Batchify adapter: wraps each individual *core.Record into a *core.RecordBatch.
	// This bridges the source (WorkerModeLoop, emits one record at a time) to the
	// batch-oriented transform stages (WorkerModeTransaction).
	batchify := &batchifyAdapter{
		ConcreteOperationExecutor: &conveyor.ConcreteOperationExecutor[*core.Record, *core.RecordBatch]{
			Name: "_batchify",
		},
	}
	if err := conveyor.AddOperation[*core.Record, *core.RecordBatch](cnv, batchify, conveyor.WorkerModeTransaction); err != nil {
		return nil, fmt.Errorf("flow %s: add batchify: %w", f.name, err)
	}

	// Transform adapters: each stage receives a *core.RecordBatch, applies the transform
	// to every record in the batch (supporting 1:N fan-out), and emits a new batch.
	// WorkerModeTransaction is correct — conveyor calls Execute once per batch item.
	for i := range f.transforms {
		t := f.transforms[i]
		ta := &batchTransformAdapter{
			ConcreteOperationExecutor: &conveyor.ConcreteOperationExecutor[*core.RecordBatch, *core.RecordBatch]{
				Name: t.Name(),
			},
			transform: t,
			ctx:       ctx,
			skipped:   &recordsSkipped,
		}
		if err := conveyor.AddOperation[*core.RecordBatch, *core.RecordBatch](cnv, ta, conveyor.WorkerModeTransaction); err != nil {
			return nil, fmt.Errorf("flow %s: add transform %s: %w", f.name, t.Name(), err)
		}
	}

	// Multi-sink adapter: writes every record in the batch to every configured sink in turn.
	ms := &multiSinkAdapter{
		ConcreteSinkExecutor: &conveyor.ConcreteSinkExecutor[*core.RecordBatch]{
			Name: "MultiSink",
		},
		sinks:   f.sinks,
		ctx:     ctx,
		counter: &recordsOut,
	}
	if err := conveyor.AddSink[*core.RecordBatch](cnv, ms, conveyor.WorkerModeTransaction); err != nil {
		return nil, fmt.Errorf("flow %s: add sink: %w", f.name, err)
	}

	pipelineErr := cnv.Start()

	stats := &core.FlowStats{
		FlowName:       f.name,
		RecordsIn:      recordsIn.Load(),
		RecordsOut:     recordsOut.Load(),
		RecordsSkipped: recordsSkipped.Load(),
		RecordsFailed:  cnv.Errors().Total(),
		ErrorsByStage:  cnv.Errors().Snapshot(),
		Duration:       time.Since(start),
	}
	if pipelineErr != nil {
		stats.Error = pipelineErr.Error()
	}
	return stats, pipelineErr
}

// ---------------------------------------------------------------------------
// Conveyor adapters
// ---------------------------------------------------------------------------

// sourceAdapter wraps core.Source as a conveyor SourceExecutor[*core.Record].
// It uses WorkerModeLoop so the entire channel is consumed by a single goroutine
// without the caller needing to manage semaphore lifecycle.
type sourceAdapter struct {
	*conveyor.ConcreteSourceExecutor[*core.Record]
	source  core.Source
	counter *atomic.Int64
}

// ExecuteLoop streams all records from the source into outChan.
// It returns as soon as the source channel is closed or ctx is cancelled.
func (a *sourceAdapter) ExecuteLoop(ctx conveyor.CnvContext, outChan chan<- *core.Record) error {
	records, err := a.source.Records(ctx)
	if err != nil {
		return fmt.Errorf("source %s: records: %w", a.source.Name(), err)
	}
	for r := range records {
		a.counter.Add(1)
		select {
		case outChan <- r:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// batchifyAdapter wraps a single *core.Record into a one-item *core.RecordBatch.
// It is the bridge from the source (which emits individual records) to the
// batch-oriented transform pipeline.
type batchifyAdapter struct {
	*conveyor.ConcreteOperationExecutor[*core.Record, *core.RecordBatch]
}

// Execute wraps r into a single-element batch.
func (a *batchifyAdapter) Execute(_ conveyor.CnvContext, r *core.Record) (*core.RecordBatch, error) {
	batch := core.RecordBatch{r}
	return &batch, nil
}

// batchTransformAdapter applies a core.Transform to every record in the input batch
// and collects all results (0..N per record) into a new output batch.
// This supports 1:N fan-out (e.g. AST parsers) while using WorkerModeTransaction.
// When a transform returns an empty result for a record, skipped is incremented.
type batchTransformAdapter struct {
	*conveyor.ConcreteOperationExecutor[*core.RecordBatch, *core.RecordBatch]
	transform core.Transform
	ctx       context.Context
	skipped   *atomic.Int64
}

// Execute processes the entire batch through the transform.
// Records for which the transform returns zero results are counted as skipped.
func (a *batchTransformAdapter) Execute(_ conveyor.CnvContext, batch *core.RecordBatch) (*core.RecordBatch, error) {
	var out core.RecordBatch
	for i := range *batch {
		results, err := a.transform.Apply(a.ctx, (*batch)[i])
		if err != nil {
			return nil, fmt.Errorf("transform %s: apply: %w", a.transform.Name(), err)
		}
		if len(results) == 0 {
			a.skipped.Add(1)
		}
		out = append(out, results...)
	}
	return &out, nil
}

// multiSinkAdapter wraps all configured sinks as a single conveyor SinkExecutor[*core.RecordBatch].
// WorkerModeTransaction is used so conveyor calls Execute once per batch.
// Every record in the batch is written to every sink in the order they were registered.
type multiSinkAdapter struct {
	*conveyor.ConcreteSinkExecutor[*core.RecordBatch]
	sinks   []core.Sink
	ctx     context.Context
	counter *atomic.Int64
}

// Execute writes every record in the batch to all sinks.
// All sinks are called for every record regardless of individual sink errors.
// Errors from all failing sinks are combined via errors.Join and returned after
// the full batch has been processed. Only records that succeeded in all sinks
// are counted toward the output counter.
func (a *multiSinkAdapter) Execute(_ conveyor.CnvContext, batch *core.RecordBatch) error {
	var errs []error
	for i := range *batch {
		r := (*batch)[i]
		var recErrs []error
		for j := range a.sinks {
			if err := a.sinks[j].Write(a.ctx, r); err != nil {
				recErrs = append(recErrs, fmt.Errorf("sink %s: write %s#%s: %w", a.sinks[j].Name(), r.Path, r.Symbol, err))
			}
		}
		if len(recErrs) == 0 {
			a.counter.Add(1)
		}
		errs = append(errs, recErrs...)
	}
	return errors.Join(errs...)
}
