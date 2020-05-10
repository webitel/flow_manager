package app

func (fm *FlowManager) JoinToInboundQueue(domainId int64, callId string, queueId int64, queueName string, priority int) (string, error) {
	return fm.cc.Member().JoinCallToQueue(domainId, callId, queueId, queueName, priority)
}
