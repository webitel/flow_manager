package call

import (
	"context"
	"fmt"
	"net/http"

	"github.com/webitel/flow_manager/flow"
	"github.com/webitel/flow_manager/model"
)

type callHandler func(ctx context.Context, scope *flow.Flow, call model.Call, args interface{}) (model.Response, *model.AppError)

func ApplicationsHandlers(r *Router) flow.ApplicationHandlers {
	var apps = make(flow.ApplicationHandlers)

	apps["ringReady"] = &flow.Application{
		Handler: callHandlerMiddleware(r.ringReady),
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
	apps["cancelQueue"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.cancelQueue),
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
	apps["tts"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.TTS),
	}
	apps["updateCid"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.UpdateCid),
	}
	apps["markIVR"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.MarkIVR),
	}
	apps["cv"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.CV),
	}
	apps["joinAgent"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.joinAgent),
	}
	apps["setGrantee"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.SetGrantee),
	}
	apps["setUser"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.SetUser),
	}
	apps["audioStream"] = &flow.Application{
		AllowNoConnect: false,
		Handler:        callHandlerMiddleware(r.audioStream),
	}

	return apps
}

func (r *Router) Request(ctx context.Context, scope *flow.Flow, req model.ApplicationRequest) <-chan model.Result {
	if h, ok := r.apps[req.Id()]; ok {
		if h.ArgsParser != nil {
			return h.Handler(ctx, scope, h.ArgsParser(scope.Connection, req.Args()))

		} else {
			return h.Handler(ctx, scope, req.Args())
		}
	} else {
		return flow.Do(func(result *model.Result) {
			result.Err = model.NewAppError("Call.Request", "call.request.not_found", nil, fmt.Sprintf("appId=%v not found", req.Id()), http.StatusNotFound)
		})
	}
}

func callHandlerMiddleware(h callHandler) flow.ApplicationHandler {
	return func(ctx context.Context, scope *flow.Flow, args interface{}) model.ResultChannel {
		return flow.Do(func(result *model.Result) {
			result.Res, result.Err = h(ctx, scope, scope.Connection.(model.Call), args)
		})
	}
}
