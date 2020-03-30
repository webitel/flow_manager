package app

func (fm *FlowManager) JoinToInboundQueue(domainId int, callId string, queueId int64, queueName string, priority int) (string, error) {
	return fm.cc.Member().JoinCallToQueue(int64(domainId), callId, queueId, queueName, priority)
}
