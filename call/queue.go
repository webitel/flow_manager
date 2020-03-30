package call

import (
	"encoding/json"
	"fmt"
	"github.com/webitel/flow_manager/model"
	"github.com/webitel/wlog"
)

/*
   {
       "joinQueue": {
           "bucket_id": null,
           "joined": [
               {
                   "sleep": "1000"
               }
           ],
           "name": "DEFAULT FROM",
           "number": "DEFAULT FROM",
           "priority": 1,
           "queue_id": 11,
           "queue_name": "INBOUND"
       }
   },
*/

type queueJoin struct {
	Name      string `json:"name"`
	Number    string `json:"number"`
	Priority  int    `json:"priority"`
	QueueId   int64  `json:"queue_id"`
	QueueName string `json:"queue_name"`
}

func (r *Router) queue(call model.Call, args interface{}) (model.Response, *model.AppError) {
	var q queueJoin
	data, _ := json.Marshal(args)
	json.Unmarshal(data, &q) //TODO

	status, err := r.fm.JoinToInboundQueue(call.DomainId(), call.Id(), q.QueueId, q.Name, q.Priority)
	if err != nil {
		wlog.Error(err.Error())
		return model.CallResponseError, nil
	}
	fmt.Println(status)

	return model.CallResponseOK, nil
}
