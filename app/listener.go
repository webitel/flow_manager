package app

// Listen starts background watchers and then blocks in the transport dispatch
// loop until all server goroutines have finished.
//
// The transport-level loop is owned by the embedded Dispatcher; this wrapper
// starts the call and list watchers first so they are alive before any
// connections arrive.
func (f *FlowManager) Listen() {
	f.callWatcher.Start(f.stop)
	f.listWatcher.Start()
	f.Dispatcher.Listen()
}
