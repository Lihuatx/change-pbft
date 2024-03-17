package network

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"simple_pbft/pbft/consensus"
	"sync"
	"time"
)

type Node struct {
	NodeID        string
	NodeTable     map[string]string // key=nodeID, value=url
	View          *View
	CurrentState  *consensus.State
	CommittedMsgs []*consensus.RequestMsg // kinda block.
	MsgBuffer     *MsgBuffer
	MsgEntrance   chan interface{}
	MsgDelivery   chan interface{}
	Alarm         chan bool
	// 请求消息的锁
	ReqMsgBufLock    sync.Mutex
	PrepreMsgBufLock sync.Mutex

	//RSA私钥
	rsaPrivKey []byte
	//RSA公钥
	rsaPubKey []byte
}

type MsgBuffer struct {
	ReqMsgs        []*consensus.RequestMsg
	PrePrepareMsgs []*consensus.PrePrepareMsg
	PrepareMsgs    []*consensus.VoteMsg
	CommitMsgs     []*consensus.VoteMsg
}

type View struct {
	ID      int64
	Primary string
}

const ResolvingTimeDuration = time.Millisecond * 1000 // 1 second.

func NewNode(nodeID string) *Node {
	const viewID = 10000000000 // temporary.

	node := &Node{
		// Hard-coded for test.
		NodeID: nodeID,
		NodeTable: map[string]string{
			"N0": "localhost:1111",
			"N1": "localhost:1112",
			"N2": "localhost:1113",
			"N3": "localhost:1114",
			"N4": "localhost:1115",
		},
		View: &View{
			ID:      viewID,
			Primary: "N0",
		},

		// Consensus-related struct
		CurrentState:  nil,
		CommittedMsgs: make([]*consensus.RequestMsg, 0),
		MsgBuffer: &MsgBuffer{
			ReqMsgs:        make([]*consensus.RequestMsg, 0),
			PrePrepareMsgs: make([]*consensus.PrePrepareMsg, 0),
			PrepareMsgs:    make([]*consensus.VoteMsg, 0),
			CommitMsgs:     make([]*consensus.VoteMsg, 0),
		},

		// Channels
		MsgEntrance: make(chan interface{}, 5),
		MsgDelivery: make(chan interface{}, 5),
		Alarm:       make(chan bool),
	}

	node.rsaPubKey = node.getPubKey(nodeID)
	node.rsaPrivKey = node.getPivKey(nodeID)

	// Start message dispatcher
	go node.dispatchMsg()

	// Start alarm trigger
	go node.alarmToDispatcher()

	// Start message resolver
	go node.resolveMsg()

	return node
}

func (node *Node) Broadcast(msg interface{}, path string) map[string]error {
	errorMap := make(map[string]error)

	for nodeID, url := range node.NodeTable {
		if nodeID == node.NodeID {
			continue
		}

		jsonMsg, err := json.Marshal(msg)
		if err != nil {
			errorMap[nodeID] = err
			continue
		}

		send(url+path, jsonMsg)
	}

	if len(errorMap) == 0 {
		return nil
	} else {
		return errorMap
	}
}

func (node *Node) Reply(msg *consensus.ReplyMsg) error {
	// Print all committed messages.
	for _, value := range node.CommittedMsgs {
		fmt.Printf("Committed value: %s, %d, %s, %d", value.ClientID, value.Timestamp, value.Operation, value.SequenceID)
	}
	fmt.Print("\n")

	node.View.ID++

	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	testTime := time.Now()
	fmt.Printf("程序运行了 %s\n", testTime)

	// Client가 없으므로, 일단 Primary에게 보내는 걸로 처리.
	send(node.NodeTable[node.View.Primary]+"/reply", jsonMsg)
	send("127.0.0.1:5000/reply", jsonMsg)

	return nil
}

// GetReq can be called when the node's CurrentState is nil.
// Consensus start procedure for the Primary.
func (node *Node) GetReq(reqMsg *consensus.RequestMsg, goOn bool) error {
	LogMsg(reqMsg)

	// Create a new state for the new consensus.
	err := node.createStateForNewConsensus(goOn)
	if err != nil {
		return err
	}

	// Start the consensus process.
	prePrepareMsg, err := node.CurrentState.StartConsensus(reqMsg)
	if err != nil {
		return err
	}

	// 主节点对消息摘要进行签名
	digestByte, _ := hex.DecodeString(prePrepareMsg.Digest)
	signInfo := node.RsaSignWithSha256(digestByte, node.rsaPrivKey)
	prePrepareMsg.Sign = signInfo

	LogStage(fmt.Sprintf("Consensus Process (ViewID:%d)", node.CurrentState.ViewID), false)

	// Send getPrePrepare message
	if prePrepareMsg != nil {
		// 附加主节点ID,用于数字签名验证
		prePrepareMsg.NodeID = node.NodeID

		node.Broadcast(prePrepareMsg, "/preprepare")
		LogStage("Pre-prepare", true)
	}

	return nil
}

// GetPrePrepare can be called when the node's CurrentState is nil.
// Consensus start procedure for normal participants.
func (node *Node) GetPrePrepare(prePrepareMsg *consensus.PrePrepareMsg, goOn bool) error {
	LogMsg(prePrepareMsg)
	fmt.Printf("-------- View ID : %d---------\n", node.View.ID)
	// Create a new state for the new consensus.
	err := node.createStateForNewConsensus(goOn)
	if err != nil {
		return err
	}
	// fmt.Printf("get Pre\n")
	digest, _ := hex.DecodeString(prePrepareMsg.Digest)
	if !node.RsaVerySignWithSha256(digest, prePrepareMsg.Sign, node.getPubKey(prePrepareMsg.NodeID)) {
		fmt.Println("节点签名验证失败！,拒绝执行Preprepare")
	}
	prePareMsg, err := node.CurrentState.PrePrepare(prePrepareMsg)
	if err != nil {
		return err
	}

	if prePareMsg != nil {
		// Attach node ID to the message 同时对摘要签名
		prePareMsg.NodeID = node.NodeID
		signInfo := node.RsaSignWithSha256(digest, node.rsaPrivKey)
		prePareMsg.Sign = signInfo

		LogStage("Pre-prepare", true)
		node.Broadcast(prePareMsg, "/prepare")
		LogStage("Prepare", false)
	}

	return nil
}

func (node *Node) GetPrepare(prepareMsg *consensus.VoteMsg) error {
	LogMsg(prepareMsg)

	digest, _ := hex.DecodeString(prepareMsg.Digest)
	if !node.RsaVerySignWithSha256(digest, prepareMsg.Sign, node.getPubKey(prepareMsg.NodeID)) {
		fmt.Println("节点签名验证失败！,拒绝执行prepare")
	}

	commitMsg, err := node.CurrentState.Prepare(prepareMsg)
	if err != nil {
		ErrMessage(prepareMsg)
		return err
	}
	if commitMsg != nil {
		// Attach node ID to the message 同时对摘要签名
		commitMsg.NodeID = node.NodeID
		signInfo := node.RsaSignWithSha256(digest, node.rsaPrivKey)
		commitMsg.Sign = signInfo

		LogStage("Prepare", true)
		node.Broadcast(commitMsg, "/commit")
		LogStage("Commit", false)
	}

	return nil
}

func (node *Node) GetCommit(commitMsg *consensus.VoteMsg) error {
	// 当节点已经完成Committed阶段后就停止接收其他节点的Committed消息
	if node.CurrentState.CurrentStage == consensus.Committed {
		return nil
	}

	LogMsg(commitMsg)

	digest, _ := hex.DecodeString(commitMsg.Digest)
	if !node.RsaVerySignWithSha256(digest, commitMsg.Sign, node.getPubKey(commitMsg.NodeID)) {
		fmt.Println("节点签名验证失败！,拒绝执行commit")
	}

	replyMsg, committedMsg, err := node.CurrentState.Commit(commitMsg)
	if err != nil {
		ErrMessage(committedMsg)
		return err
	}

	if replyMsg != nil {
		if committedMsg == nil {
			return errors.New("committed message is nil, even though the reply message is not nil")
		}

		// Attach node ID to the message
		replyMsg.NodeID = node.NodeID

		// Save the last version of committed messages to node.
		node.CommittedMsgs = append(node.CommittedMsgs, committedMsg)

		LogStage("Commit", true)
		node.Reply(replyMsg)
		LogStage("Reply\n", true)

		//	对于主节点而言，如果请求缓存池中还有请求，需要继续执行本地共识
		if node.NodeID == node.View.Primary {
			var TempReqMsg *consensus.RequestMsg
			node.ReqMsgBufLock.Lock()
			if len(node.MsgBuffer.ReqMsgs) > 0 {
				// 直接获取第一个请求消息
				TempReqMsg = node.MsgBuffer.ReqMsgs[0]
				// 直接更新请求消息缓冲区，去掉已处理的第一个消息
				node.MsgBuffer.ReqMsgs = node.MsgBuffer.ReqMsgs[1:]
			}
			node.ReqMsgBufLock.Unlock()

			// 如果有请求消息，则继续执行相关处理
			if TempReqMsg != nil {
				fmt.Printf("                                  go on                               go on\n")
				node.GetReq(TempReqMsg, true)
			} else {
				node.CurrentState.CurrentStage = consensus.Committed
			}
		} else { // 如果已经有 Preprepare 缓存消息
			var TempReqMsg *consensus.PrePrepareMsg
			node.PrepreMsgBufLock.Lock()
			if len(node.MsgBuffer.PrePrepareMsgs) > 0 {
				// 直接获取第一个请求消息
				TempReqMsg = node.MsgBuffer.PrePrepareMsgs[0]
				// 直接更新请求消息缓冲区，去掉已处理的第一个消息
				node.MsgBuffer.PrePrepareMsgs = node.MsgBuffer.PrePrepareMsgs[1:]
			}
			node.PrepreMsgBufLock.Unlock()

			// 如果有请求消息，则继续执行相关处理
			if TempReqMsg != nil {
				fmt.Printf("                                  go on                               go on\n")
				node.GetPrePrepare(TempReqMsg, true)
			} else {
				node.CurrentState.CurrentStage = consensus.Committed
			}
		}

	}

	return nil
}

func (node *Node) GetReply(msg *consensus.ReplyMsg) {
	fmt.Printf("Result: %s by %s\n", msg.Result, msg.NodeID)
}

func (node *Node) createStateForNewConsensus(goOn bool) error {
	// Check if there is an ongoing consensus process.
	if node.CurrentState != nil {
		if node.CurrentState.CurrentStage != consensus.Committed && !goOn {
			return errors.New("another consensus is ongoing")
		}
	}

	// Get the last sequence ID
	var lastSequenceID int64
	if len(node.CommittedMsgs) == 0 {
		lastSequenceID = -1
	} else {
		lastSequenceID = node.CommittedMsgs[len(node.CommittedMsgs)-1].SequenceID
	}

	// Create a new state for this new consensus process in the Primary
	node.CurrentState = consensus.CreateState(node.View.ID, lastSequenceID)

	LogStage("Create the replica status", true)

	return nil
}

func (node *Node) dispatchMsg() {
	for {
		select {
		case msg := <-node.MsgEntrance:
			err := node.routeMsg(msg)
			if err != nil {
				fmt.Println(err)
				// TODO: send err to ErrorChannel
			}
		case <-node.Alarm:
			err := node.routeMsgWhenAlarmed()
			if err != nil {
				fmt.Println(err)
				// TODO: send err to ErrorChannel
			}
		}
	}
}

func (node *Node) routeMsg(msg interface{}) []error {
	switch msg.(type) {
	case *consensus.RequestMsg:
		if node.CurrentState == nil || node.CurrentState.CurrentStage == consensus.Committed { //一开始没有进行共识的时候，此时 currentstate 为nil
			// Copy buffered messages first.
			msgs := make([]*consensus.RequestMsg, len(node.MsgBuffer.ReqMsgs))
			copy(msgs, node.MsgBuffer.ReqMsgs)

			// Append a newly arrived message.
			msgs = append(msgs, msg.(*consensus.RequestMsg))

			// Empty the buffer.
			node.ReqMsgBufLock.Lock()
			node.MsgBuffer.ReqMsgs = make([]*consensus.RequestMsg, 0)
			node.ReqMsgBufLock.Unlock()
			// Send messages.
			node.MsgDelivery <- msgs
		} else {
			node.ReqMsgBufLock.Lock()
			node.MsgBuffer.ReqMsgs = append(node.MsgBuffer.ReqMsgs, msg.(*consensus.RequestMsg))
			node.ReqMsgBufLock.Unlock()
		}
		fmt.Printf("                    request buffer %d\n", len(node.MsgBuffer.ReqMsgs))
	case *consensus.PrePrepareMsg:
		if node.CurrentState == nil || node.CurrentState.CurrentStage == consensus.Committed {
			// Copy buffered messages first.
			node.PrepreMsgBufLock.Lock()
			msgs := make([]*consensus.PrePrepareMsg, len(node.MsgBuffer.PrePrepareMsgs))
			copy(msgs, node.MsgBuffer.PrePrepareMsgs)

			// Append a newly arrived message.
			msgs = append(msgs, msg.(*consensus.PrePrepareMsg))

			// Empty the buffer.
			node.MsgBuffer.PrePrepareMsgs = make([]*consensus.PrePrepareMsg, 0)
			node.PrepreMsgBufLock.Unlock()

			// Send messages.
			node.MsgDelivery <- msgs
		} else {
			node.PrepreMsgBufLock.Lock()
			node.MsgBuffer.PrePrepareMsgs = append(node.MsgBuffer.PrePrepareMsgs, msg.(*consensus.PrePrepareMsg))
			node.PrepreMsgBufLock.Unlock()
		}
	case *consensus.VoteMsg:
		if msg.(*consensus.VoteMsg).MsgType == consensus.PrepareMsg {
			// if node.CurrentState == nil || node.CurrentState.CurrentStage != consensus.PrePrepared
			// 这样的写法会导致当当前节点已经收到2f个节点进入committed阶段时，就会把后来收到的Preprepare消息放到缓冲区中，
			// 这样在下次共识又到prePrepare阶段时就会先去处理上一轮共识的prePrepare协议！
			if node.CurrentState == nil || node.CurrentState.CurrentStage != consensus.PrePrepared {
				node.MsgBuffer.PrepareMsgs = append(node.MsgBuffer.PrepareMsgs, msg.(*consensus.VoteMsg))
			} else {
				// Copy buffered messages first.
				var msgs []*consensus.VoteMsg
				var msgSave []*consensus.VoteMsg
				node.MsgBuffer.PrepareMsgs = append(node.MsgBuffer.PrepareMsgs, msg.(*consensus.VoteMsg))
				copy(msgs, node.MsgBuffer.PrepareMsgs)

				for _, value := range node.MsgBuffer.PrepareMsgs {
					if value.ViewID == node.View.ID {
						msgs = append(msgs, value)
					} else if value.ViewID > node.View.ID {
						msgSave = append(msgSave, value)
					}

				}
				// Append a newly arrived message.
				// msgs = append(msgs, msg.(*consensus.VoteMsg))
				// Empty the buffer.
				node.MsgBuffer.PrepareMsgs = msgSave

				// Send messages.
				node.MsgDelivery <- msgs

			}
		} else if msg.(*consensus.VoteMsg).MsgType == consensus.CommitMsg {
			if node.CurrentState == nil || node.CurrentState.CurrentStage != consensus.Prepared {
				node.MsgBuffer.CommitMsgs = append(node.MsgBuffer.CommitMsgs, msg.(*consensus.VoteMsg))
			} else {
				// Copy buffered messages first.
				var msgs []*consensus.VoteMsg
				var msgSave []*consensus.VoteMsg
				node.MsgBuffer.CommitMsgs = append(node.MsgBuffer.CommitMsgs, msg.(*consensus.VoteMsg))
				copy(msgs, node.MsgBuffer.CommitMsgs)

				for _, value := range node.MsgBuffer.CommitMsgs {
					if value.ViewID == node.View.ID {
						msgs = append(msgs, value)
					} else if value.ViewID > node.View.ID {
						msgSave = append(msgSave, value)
					}

				}
				// Append a newly arrived message.
				// msgs = append(msgs, msg.(*consensus.VoteMsg))
				// Empty the buffer.
				node.MsgBuffer.CommitMsgs = msgSave

				// Send messages.
				node.MsgDelivery <- msgs
			}
		}

	}

	return nil
}

func (node *Node) routeMsgWhenAlarmed() []error {
	if len(node.MsgEntrance) == 5 {
		fmt.Printf("MsgEntrance chan is Fulled\n")
	}
	if len(node.MsgDelivery) == 5 {
		fmt.Printf("MsgDeliveru chan is Fulled\n")
	}
	if node.CurrentState == nil || node.CurrentState.CurrentStage == consensus.Committed {
		// Check ReqMsgs, send them.
		if len(node.MsgBuffer.ReqMsgs) != 0 {
			msgs := make([]*consensus.RequestMsg, len(node.MsgBuffer.ReqMsgs))
			copy(msgs, node.MsgBuffer.ReqMsgs)

			node.MsgDelivery <- msgs
			fmt.Printf("[Alarm]--node.MsgBuffer.ReqMsgs\n")
		}

		// Check PrePrepareMsgs, send them.
		if len(node.MsgBuffer.PrePrepareMsgs) != 0 {
			msgs := make([]*consensus.PrePrepareMsg, len(node.MsgBuffer.PrePrepareMsgs))
			copy(msgs, node.MsgBuffer.PrePrepareMsgs)
			fmt.Printf("[Alarm]--node.MsgBuffer.PrePrepareMsgs\n")
			for _, value := range msgs {
				fmt.Printf("View ID %d", value.ViewID)
			}
			node.MsgDelivery <- msgs
		}
	} else {
		switch node.CurrentState.CurrentStage {
		case consensus.PrePrepared:
			// Check PrepareMsgs, send them.
			if len(node.MsgBuffer.PrepareMsgs) != 0 {
				msgs := make([]*consensus.VoteMsg, len(node.MsgBuffer.PrepareMsgs))
				copy(msgs, node.MsgBuffer.PrepareMsgs)
				fmt.Printf("[Alarm]--node.MsgBuffer.prepareMsgs\n")

				node.MsgDelivery <- msgs
			}
		case consensus.Prepared:
			// Check CommitMsgs, send them.
			if len(node.MsgBuffer.CommitMsgs) != 0 {
				msgs := make([]*consensus.VoteMsg, len(node.MsgBuffer.CommitMsgs))
				copy(msgs, node.MsgBuffer.CommitMsgs)
				fmt.Printf("[Alarm]--node.MsgBuffer.CommitMsgs\n")

				node.MsgDelivery <- msgs
			}
		}
	}

	return nil
}

func (node *Node) resolveMsg() {
	for {
		// Get buffered messages from the dispatcher.
		msgs := <-node.MsgDelivery
		switch msgs.(type) {
		case []*consensus.RequestMsg:
			errs := node.resolveRequestMsg(msgs.([]*consensus.RequestMsg))
			if len(errs) != 0 {
				for _, err := range errs {
					fmt.Println(err)
				}
				// TODO: send err to ErrorChannel
			}
		case []*consensus.PrePrepareMsg:
			errs := node.resolvePrePrepareMsg(msgs.([]*consensus.PrePrepareMsg))
			if len(errs) != 0 {
				for _, err := range errs {
					fmt.Println(err)
				}
				// TODO: send err to ErrorChannel
			}
		case []*consensus.VoteMsg:
			voteMsgs := msgs.([]*consensus.VoteMsg)
			if len(voteMsgs) == 0 {
				break
			}

			if voteMsgs[0].MsgType == consensus.PrepareMsg {
				errs := node.resolvePrepareMsg(voteMsgs)
				if len(errs) != 0 {
					for _, err := range errs {
						fmt.Println(err)
					}
					// TODO: send err to ErrorChannel
				}
			} else if voteMsgs[0].MsgType == consensus.CommitMsg {
				errs := node.resolveCommitMsg(voteMsgs)
				if len(errs) != 0 {
					for _, err := range errs {
						fmt.Println(err)
					}
					// TODO: send err to ErrorChannel
				}

			}
		}
	}
}

func (node *Node) alarmToDispatcher() {
	for {
		time.Sleep(ResolvingTimeDuration)
		node.Alarm <- true
	}
}

func (node *Node) resolveRequestMsg(msgs []*consensus.RequestMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	fmt.Printf("len RequestMsg msg %d\n", len(msgs))

	err := node.GetReq(msgs[0], false)
	if err != nil {
		return errs
	}

	if len(msgs) > 1 {
		node.ReqMsgBufLock.Lock()
		tempMsg := msgs[1:]
		TempReqBuf := make([]*consensus.RequestMsg, len(node.MsgBuffer.ReqMsgs))
		copy(TempReqBuf, node.MsgBuffer.ReqMsgs)
		// Append a newly arrived message.
		tempMsg = append(tempMsg, TempReqBuf...)
		node.MsgBuffer.ReqMsgs = tempMsg
		node.ReqMsgBufLock.Unlock()
	}

	return nil
}

func (node *Node) resolvePrePrepareMsg(msgs []*consensus.PrePrepareMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	// 从下标num_of_event_to_resolve开始执行，之前执行过的PrePrepareMsg不需要再执行
	fmt.Printf("len PrePrepareMsg msg %d\n", len(msgs))

	for _, prePrepareMsg := range msgs {
		if prePrepareMsg.ViewID < node.View.ID {
			continue
		}
		err := node.GetPrePrepare(prePrepareMsg, false)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) resolvePrepareMsg(msgs []*consensus.VoteMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	fmt.Printf("len PrepareMsg msg %d\n", len(msgs))
	for _, prepareMsg := range msgs {
		if prepareMsg.ViewID < node.View.ID {
			continue
		}
		err := node.GetPrepare(prepareMsg)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) resolveCommitMsg(msgs []*consensus.VoteMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	fmt.Printf("len CommitMsg msg %d node\n", len(msgs))

	for _, commitMsg := range msgs {
		if commitMsg.ViewID < node.View.ID {
			continue
		} else if commitMsg.ViewID > node.View.ID {

		}
		err := node.GetCommit(commitMsg)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

// 传入节点编号， 获取对应的公钥
func (node *Node) getPubKey(nodeID string) []byte {
	key, err := ioutil.ReadFile("Keys/" + nodeID + "/" + nodeID + "_RSA_PUB")
	if err != nil {
		log.Panic(err)
	}
	return key
}

// 传入节点编号， 获取对应的私钥
func (node *Node) getPivKey(nodeID string) []byte {
	key, err := ioutil.ReadFile("Keys/" + nodeID + "/" + nodeID + "_RSA_PIV")
	if err != nil {
		log.Panic(err)
	}
	return key
}

// 数字签名
func (node *Node) RsaSignWithSha256(data []byte, keyBytes []byte) []byte {
	h := sha256.New()
	h.Write(data)
	hashed := h.Sum(nil)
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		panic(errors.New("private key error"))
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		fmt.Println("ParsePKCS8PrivateKey err", err)
		panic(err)
	}

	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed)
	if err != nil {
		fmt.Printf("Error from signing: %s\n", err)
		panic(err)
	}

	return signature
}

// 签名验证
func (node *Node) RsaVerySignWithSha256(data, signData, keyBytes []byte) bool {
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		panic(errors.New("public key error"))
	}
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		panic(err)
	}

	hashed := sha256.Sum256(data)
	err = rsa.VerifyPKCS1v15(pubKey.(*rsa.PublicKey), crypto.SHA256, hashed[:], signData)
	if err != nil {
		panic(err)
	}
	return true
}
