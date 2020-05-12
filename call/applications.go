package call

import (
	"context"
	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
	"net/http"
)

type callHandler func(call model.Call, args interface{}) (model.Response, *model.AppError)

func ApplicationsHandlers(r *Router) flow.ApplicationHandlers {
	var apps = make(flow.ApplicationHandlers)

	apps["ringReady"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.ringReady),
	}
	apps["preAnswer"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.preAnswer),
	}
	apps["answer"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.answer),
	}
	apps["hangup"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.hangup),
	}
	apps["setAll"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.setAll),
	}
	apps["setNoLocal"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.setNoLocal),
	}
	apps["unSet"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.UnSet),
	}
	apps["bridge"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.bridge),
	}
	apps["echo"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.echo),
	}
	apps["export"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.export),
	}
	apps["recordFile"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.recordFile),
	}
	apps["recordSession"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.recordSession),
	}
	apps["sleep"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.sleep),
	}
	apps["conference"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.conference),
	}
	apps["joinQueue"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.queue),
	}
	apps["flushDtmf"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.dtmfFlush),
	}
	apps["inBandDTMF"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.inBandDTMF),
	}
	apps["park"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.park),
	}
	apps["sipRedirect"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.SipRedirect),
	}
	apps["playback"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.Playback),
	}
	apps["ringback"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.RingBack),
	}
	apps["setSounds"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.SetSounds),
	}
	apps["scheduleHangup"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.ScheduleHangup),
	}

	return apps
}

func callHandlerMiddleware(h callHandler) flow.ApplicationHandler {
	return func(ctx context.Context, c model.Connection, args interface{}) (model.Response, *model.AppError) {
		if c.Type() != model.ConnectionTypeCall {
			return nil, model.NewAppError("Call", "call.middleware.valid.type", nil, "bad type", http.StatusBadRequest)
		}
		return h(c.(model.Call), args)
	}
}
