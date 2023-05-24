package bpmn

type SequenceFlow struct {
	ModelMeta
	In        string
	Out       string
	Condition *ConditionExpression
}

type ConditionExpression struct {
	Type       string
	Expression string
}
