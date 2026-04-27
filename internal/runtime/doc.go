// Package runtime implements a resumable flow interpreter for chat, IM, and
// email channels. It runs alongside the legacy flow/ package (which handles
// call/webhook/processing and continues unchanged).
//
// Mental model:
//
//	schema JSON ─► tree.Parser ─► tree.Tree
//	                                   │
//	                         interpreter.Step(ctx, log, state, tree, ops)
//	                                   │
//	                         state.ExecState  (serializable, stored in DB)
//
// The state.ExecState is a plain serializable value — no goroutines, no
// channels, no pointers. Suspend/resume is implemented by persisting the
// state before blocking operations and reloading it on restart.
//
// Package boundaries:
//   - internal/runtime/state  — pure data types, no I/O
//   - internal/runtime/tree   — parsed schema, no execution
//   - internal/runtime/interpreter — executor, depends on state+tree+ops
//   - internal/runtime/ops    — Op registry + builtin + legacy bridge
//   - internal/runtime/persistence — port (interface), no implementation
package runtime
