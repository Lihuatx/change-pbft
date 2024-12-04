package consensus

type RequestMsg struct {
	Timestamp  int64  `json:"timestamp"`
	ClientID   string `json:"clientID"`
	Operation  string `json:"operation"`
	SequenceID int64  `json:"sequenceID"`
}

type BatchRequestMsg struct {
	Requests  [BatchSize]*RequestMsg `json:"Requests"`
	Timestamp int64                  `json:"timestamp"`
	ClientID  string                 `json:"clientID"`
}

type ReplyMsg struct {
	ViewID    int64  `json:"viewID"`
	Timestamp int64  `json:"timestamp"`
	ClientID  string `json:"clientID"`
	NodeID    string `json:"nodeID"`
	Result    string `json:"result"`
}

type PrePrepareMsg struct {
	ViewID     int64            `json:"viewID"`
	SequenceID int64            `json:"sequenceID"`
	Digest     string           `json:"digest"`
	NodeID     string           `json:"nodeID"` //添加nodeID
	RequestMsg *BatchRequestMsg `json:"requestMsg"`
	Sign       []byte           //添加sign
}

type VoteMsg struct {
	ViewID     int64  `json:"viewID"`
	SequenceID int64  `json:"sequenceID"`
	Digest     string `json:"digest"`
	NodeID     string `json:"nodeID"`
	MsgType    `json:"msgType"`
	Sign       []byte //添加sign
}

type MsgType int

const BatchSize = 1

const (
	PrepareMsg MsgType = iota
	CommitMsg
)
