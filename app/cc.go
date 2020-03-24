package app

func (fm *FlowManager) JoinToInboundQueue(domainId int, callId string, queueid int64, queueName string, priority int) {
	//fm.cc.Agent().Pause()
	fm.cc.Member().JoinCallToQueue(int64(domainId), callId, queueid, queueName, priority)
}
