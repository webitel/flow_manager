package flow

import (
	"fmt"

	"github.com/robertkrimen/otto"
	"github.com/webitel/wlog"
)

func (f *Flow) initVm() {
	f.vm = otto.New()

	f.vm.Interrupt = make(chan func(), 1) // The buffer prevents blocking

	err := f.vm.Set("_getChannelVar", func(call otto.FunctionCall) otto.Value {
		if f.Connection == nil {
			// error
			return otto.Value{}
		}

		v, _ := f.Connection.Get(call.Argument(0).String())
		res, err := f.vm.ToValue(v)
		if err != nil {
			return otto.Value{}
		}
		return res
	})
	if err != nil {
		wlog.Error(fmt.Sprintf("connection init VM error: %s", err.Error()))
	}

	err = f.vm.Set("_LocalDateParameters", func(call otto.FunctionCall) otto.Value {
		t := f.Now()
		res, err := f.vm.ToValue([]int{t.Year(), int(t.Month()), t.Day(), t.Hour(), t.Minute(), t.Second()})
		if err != nil {
			return otto.Value{}
		}

		return res
	})
	if err != nil {
		wlog.Error(fmt.Sprintf("connection init VM error: %s", err.Error()))
	}

}

func (f *Flow) GetVm() *otto.Otto {
	f.Lock()
	defer f.Unlock()

	if f.vm != nil {
		return f.vm
	}

	f.initVm()
	return f.vm
}
