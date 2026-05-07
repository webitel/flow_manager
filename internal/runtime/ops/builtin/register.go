package builtin

import "github.com/webitel/flow_manager/internal/runtime/ops"

// Register adds all builtin ops to reg. Call this once during application
// bootstrap before creating any Driver.
func Register(reg *ops.Registry) {
	reg.Register("execute", Execute())
	reg.Register("if", If())
	reg.Register("while", While())
	reg.Register("switch", Switch())
	reg.Register("goto", Goto())
	reg.Register("break", Break())
	reg.Register("set", Set())
	reg.Register("log", Log())
	reg.Register("softSleep", SoftSleep())
	reg.Register("string", StringOp())
	reg.Register("math", MathOp())
	reg.Register("classifier", Classifier())
	reg.Register("dump", Dump())
}
