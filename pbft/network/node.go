package network

import (
	"bufio"
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
	"os"
	"simple_pbft/pbft/consensus"
	"strings"
	"sync"
	"time"
)

type Node struct {
	NodeID         string
	NodeTable      map[string]string // key=nodeID, value=url
	View           *View
	CurrentState   *consensus.State
	CommittedMsgs  []*consensus.RequestMsg // kinda block.
	MsgBuffer      *MsgBuffer
	MsgEntrance    chan interface{}
	MsgDelivery    chan interface{}
	MsgRequsetchan chan interface{}
	Alarm          chan bool
	// 请求消息的锁
	MsgBufferLock *MsgBufferLock

	//RSA私钥
	rsaPrivKey []byte
	//RSA公钥
	rsaPubKey []byte
}

type MsgBuffer struct {
	ReqMsgs        []*consensus.RequestMsg
	BatchReqMsgs   []*consensus.BatchRequestMsg
	PrePrepareMsgs []*consensus.PrePrepareMsg
	PrepareMsgs    []*consensus.VoteMsg
	CommitMsgs     []*consensus.VoteMsg
}

type View struct {
	ID      int64
	Primary string
}

type MsgBufferLock struct {
	ReqMsgsLock        sync.Mutex
	PrePrepareMsgsLock sync.Mutex
	PrepareMsgsLock    sync.Mutex
	CommitMsgsLock     sync.Mutex
}

const ResolvingTimeDuration = time.Millisecond * 1000 // 1 second.

func NewNode(nodeID string) *Node {
	const viewID = 10000000000 // temporary.
	node := &Node{
		// Hard-coded for test.
		NodeID: nodeID,

		View: &View{
			ID:      viewID,
			Primary: "N0",
		},

		// Consensus-related struct
		CurrentState:  nil,
		CommittedMsgs: make([]*consensus.RequestMsg, 0),
		MsgBuffer: &MsgBuffer{
			ReqMsgs:        make([]*consensus.RequestMsg, 0),
			BatchReqMsgs:   make([]*consensus.BatchRequestMsg, 0),
			PrePrepareMsgs: make([]*consensus.PrePrepareMsg, 0),
			PrepareMsgs:    make([]*consensus.VoteMsg, 0),
			CommitMsgs:     make([]*consensus.VoteMsg, 0),
		},
		MsgBufferLock: &MsgBufferLock{},
		// Channels
		MsgEntrance:    make(chan interface{}, 200),
		MsgDelivery:    make(chan interface{}, 200),
		MsgRequsetchan: make(chan interface{}, 200),
		Alarm:          make(chan bool),
	}

	node.NodeTable = LoadNodeTable("nodetable.txt")

	node.rsaPubKey = node.getPubKey(nodeID)
	node.rsaPrivKey = node.getPivKey(nodeID)
	node.CurrentState = consensus.CreateState(node.View.ID, -2)
	// 专门用于收取客户端请求,防止堵塞其他线程
	go node.resolveClientRequest()

	// Start message dispatcher
	go node.dispatchMsg()

	// Start alarm trigger
	go node.alarmToDispatcher()

	// Start message resolver
	go node.resolveMsg()

	return node
}

// LoadNodeTable 从指定的文件路径加载 NodeTable
func LoadNodeTable(filePath string) map[string]string {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	// 初始化 NodeTable
	nodeTable := make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) == 2 {
			nodeID, address := parts[0], parts[1]
			nodeTable[nodeID] = address
		}
	}

	if err := scanner.Err(); err != nil {
		return nil
	}

	return nodeTable
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

		//fmt.Printf("Broadcasting to %s, message size: %d bytes\n", nodeID, len(jsonMsg))

		send(url+path, jsonMsg)
	}

	if len(errorMap) == 0 {
		return nil
	} else {
		return errorMap
	}
}

var start time.Time
var duration time.Duration

func (node *Node) Reply(msg *consensus.ReplyMsg) error {
	// Print all committed messages.
	//for _, value := range node.CommittedMsgs {
	//	///fmt.Printf("Committed value: %s, %d, %s, %d", value.ClientID, value.Timestamp, value.Operation, value.SequenceID)
	//}
	///fmt.Print("\n")
	const viewID = 10000000000
	node.View.ID++
	fmt.Printf("View ID: %d\n", node.View.ID)

	if len(node.CommittedMsgs) == 1 && node.NodeID == node.View.Primary {
		//start = time.Now()
	} else if len(node.CommittedMsgs) == 300 && node.NodeID == node.View.Primary {
		duration = time.Since(start)
		// 打开文件，如果文件不存在则创建，如果文件存在则追加内容
		fmt.Printf("Function took %s\n", duration)
		file, err := os.OpenFile("example.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		// 使用fmt.Fprintf格式化写入内容到文件
		_, err = fmt.Fprintf(file, "durtion: %s\n", duration)
		if err != nil {
			log.Fatal(err)
		}

	} else if len(node.CommittedMsgs) > 300 && node.NodeID == node.View.Primary {
		fmt.Printf("  Function took %s\n", duration)
	}
	if node.NodeID == node.View.Primary {
		go func() {
			for i := consensus.BatchSize; i > 0; i-- {
				replyClientMsg := node.CommittedMsgs[len(node.CommittedMsgs)-i]
				jsonMsg, _ := json.Marshal(replyClientMsg)
				// 系统中没有设置用户，reply消息直接发送给主节点
				url := ClientURL["N"] + "/reply"
				send(url, jsonMsg)
				fmt.Printf("\n\nReply to Client!\n\n\n")
			}
		}()
	}

	return nil
}

// GetReq can be called when the node's CurrentState is nil.
// Consensus start procedure for the Primary.
func (node *Node) GetReq(reqMsg *consensus.BatchRequestMsg, goOn bool) error {
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
	// ///fmt.Printf("get Pre\n")
	digest, _ := hex.DecodeString(prePrepareMsg.Digest)
	if !node.RsaVerySignWithSha256(digest, prePrepareMsg.Sign, node.getPubKey(prePrepareMsg.NodeID)) {
		///fmt.Println("节点签名验证失败！,拒绝执行Preprepare")
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
		///fmt.Println("节点签名验证失败！,拒绝执行prepare")
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
		///fmt.Println("节点签名验证失败！,拒绝执行commit")
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
		for i := 0; i < consensus.BatchSize; i++ {
			node.CommittedMsgs = append(node.CommittedMsgs, committedMsg.Requests[i])
		}

		LogStage("Commit", true)
		node.Reply(replyMsg)
		LogStage("Reply\n", true)
		node.CurrentState.CurrentStage = consensus.Committed
		//	对于主节点而言，如果请求缓存池中还有请求，需要继续执行本地共识

	}

	return nil
}

func (node *Node) GetReply(msg *consensus.ReplyMsg) {
	///fmt.Printf("Result: %s by %s\n", msg.Result, msg.NodeID)
}

func (node *Node) createStateForNewConsensus(goOn bool) error {
	// Check if there is an ongoing consensus process.
	if node.CurrentState.LastSequenceID != -2 {
		if node.CurrentState.CurrentStage != consensus.Committed && !goOn && node.CurrentState.CurrentStage != consensus.GetRequest {
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
		time.Sleep(10 * time.Microsecond)

		select {
		case msg := <-node.MsgEntrance:
			//fmt.Printf("Send node.MsgEntrance  ")
			err := node.routeMsg(msg)
			if err != nil {
				fmt.Println(err)
				// TODO: send err to ErrorChannel
			}
		case <-node.Alarm:
			//fmt.Printf("a\n")
			//err := node.routeMsgWhenAlarmed()
			//if err != nil {
			//	fmt.Println(err)
			//	// TODO: send err to ErrorChannel
			//}
		}
	}
}

func (node *Node) SaveClientRequest(msg interface{}) {
	switch msg.(type) {
	case *consensus.RequestMsg:
		//一开始没有进行共识的时候，此时 currentstate 为nil
		node.MsgBufferLock.ReqMsgsLock.Lock()
		node.MsgBuffer.ReqMsgs = append(node.MsgBuffer.ReqMsgs, msg.(*consensus.RequestMsg))
		node.MsgBufferLock.ReqMsgsLock.Unlock()
		fmt.Printf("缓存中收到 %d 条客户端请求\n", len(node.MsgBuffer.ReqMsgs))
	}
}

func (node *Node) resolveClientRequest() {
	for {
		time.Sleep(10 * time.Microsecond)

		select {
		case msg := <-node.MsgRequsetchan:
			node.SaveClientRequest(msg)
			//time.Sleep(50 * time.Millisecond) // 程序暂停100毫秒
		}
	}
}

func (node *Node) routeMsg(msg interface{}) []error {
	switch msg.(type) {

	case *consensus.PrePrepareMsg:

		node.MsgBufferLock.PrePrepareMsgsLock.Lock()
		node.MsgBuffer.PrePrepareMsgs = append(node.MsgBuffer.PrePrepareMsgs, msg.(*consensus.PrePrepareMsg))
		node.MsgBufferLock.PrePrepareMsgsLock.Unlock()
		//fmt.Printf("                    Msgbuffer %d %d %d %d\n", len(node.MsgBuffer.ReqMsgs), len(node.MsgBuffer.PrePrepareMsgs), len(node.MsgBuffer.PrepareMsgs), len(node.MsgBuffer.CommitMsgs))

	case *consensus.VoteMsg:
		if msg.(*consensus.VoteMsg).MsgType == consensus.PrepareMsg {
			// if node.CurrentState == nil || node.CurrentState.CurrentStage != consensus.PrePrepared
			// 这样的写法会导致当当前节点已经收到2f个节点进入committed阶段时，就会把后来收到的Preprepare消息放到缓冲区中，
			// 这样在下次共识又到prePrepare阶段时就会先去处理上一轮共识的prePrepare协议！
			node.MsgBufferLock.PrepareMsgsLock.Lock()
			node.MsgBuffer.PrepareMsgs = append(node.MsgBuffer.PrepareMsgs, msg.(*consensus.VoteMsg))
			node.MsgBufferLock.PrepareMsgsLock.Unlock()
		} else if msg.(*consensus.VoteMsg).MsgType == consensus.CommitMsg {
			node.MsgBufferLock.CommitMsgsLock.Lock()
			node.MsgBuffer.CommitMsgs = append(node.MsgBuffer.CommitMsgs, msg.(*consensus.VoteMsg))
			node.MsgBufferLock.CommitMsgsLock.Unlock()
		}

		//fmt.Printf("                    Msgbuffer %d %d %d %d\n", len(node.MsgBuffer.ReqMsgs), len(node.MsgBuffer.PrePrepareMsgs), len(node.MsgBuffer.PrepareMsgs), len(node.MsgBuffer.CommitMsgs))
	}

	return nil
}

var lastViewId int64

func (node *Node) routeMsgWhenAlarmed() []error {
	if node.View.ID != lastViewId {
		fmt.Printf("                                                                View ID %d\n", node.View.ID)
		lastViewId = node.View.ID
	}
	if node.CurrentState.LastSequenceID == -2 || node.CurrentState.CurrentStage == consensus.Committed {
		// Check ReqMsgs, send them.
		if len(node.MsgBuffer.ReqMsgs) != 0 {
			msgs := make([]*consensus.RequestMsg, len(node.MsgBuffer.ReqMsgs))
			copy(msgs, node.MsgBuffer.ReqMsgs)
			for _, value := range msgs {
				fmt.Printf("Requset timeStamp %d", value.Timestamp)
			}
			fmt.Printf("\n")
			//node.MsgDelivery <- msgs
			///fmt.Printf("[Alarm]--node.MsgBuffer.ReqMsgs\n")
		}

		// Check PrePrepareMsgs, send them.
		if len(node.MsgBuffer.PrePrepareMsgs) != 0 {
			msgs := make([]*consensus.PrePrepareMsg, len(node.MsgBuffer.PrePrepareMsgs))
			copy(msgs, node.MsgBuffer.PrePrepareMsgs)
			///fmt.Printf("[Alarm]--node.MsgBuffer.PrePrepareMsgs\n")
			///for _, value := range msgs {
			///fmt.Printf("View ID %d", value.ViewID)
			///}
			for _, value := range msgs {
				fmt.Printf("PrePrepare View ID %d", value.ViewID)
			}
			fmt.Printf("\n")
			node.MsgDelivery <- msgs
		}
	} else {
		switch node.CurrentState.CurrentStage {
		case consensus.PrePrepared:
			// Check PrepareMsgs, send them.
			if len(node.MsgBuffer.PrepareMsgs) != 0 {
				msgs := make([]*consensus.VoteMsg, len(node.MsgBuffer.PrepareMsgs))
				copy(msgs, node.MsgBuffer.PrepareMsgs)
				///fmt.Printf("[Alarm]--node.MsgBuffer.prepareMsgs\n")
				for _, value := range msgs {
					fmt.Printf("PrepareMsgs View ID %d", value.ViewID)
				}
				fmt.Printf("\n")
				node.MsgDelivery <- msgs
			}
		case consensus.Prepared:
			// Check CommitMsgs, send them.
			if len(node.MsgBuffer.CommitMsgs) != 0 {
				msgs := make([]*consensus.VoteMsg, len(node.MsgBuffer.CommitMsgs))
				copy(msgs, node.MsgBuffer.CommitMsgs)
				///fmt.Printf("[Alarm]--node.MsgBuffer.CommitMsgs\n")
				for _, value := range msgs {
					fmt.Printf("CommitMsgs View ID %d", value.ViewID)
				}
				fmt.Printf("\n")
				node.MsgDelivery <- msgs
			}
		}
	}

	return nil
}

// 出队
// Dequeue for Request messages
func (mb *MsgBuffer) DequeueReqMsg() *consensus.RequestMsg {
	if len(mb.ReqMsgs) == 0 {
		return nil
	}
	msg := mb.ReqMsgs[0]                          // 获取第一个元素
	mb.ReqMsgs = mb.ReqMsgs[consensus.BatchSize:] // 移除第一个元素
	return msg
}

// Dequeue for PrePrepare messages
func (mb *MsgBuffer) DequeuePrePrepareMsg() *consensus.PrePrepareMsg {
	if len(mb.PrePrepareMsgs) == 0 {
		return nil
	}
	msg := mb.PrePrepareMsgs[0]
	mb.PrePrepareMsgs = mb.PrePrepareMsgs[1:]
	return msg
}

// Dequeue for Prepare messages
func (mb *MsgBuffer) DequeuePrepareMsg() *consensus.VoteMsg {
	if len(mb.PrepareMsgs) == 0 {
		return nil
	}
	msg := mb.PrepareMsgs[0]
	mb.PrepareMsgs = mb.PrepareMsgs[1:]
	return msg
}

// Dequeue for Prepare messages
func (mb *MsgBuffer) DequeueCommitMsg() *consensus.VoteMsg {
	if len(mb.CommitMsgs) == 0 {
		return nil
	}
	msg := mb.CommitMsgs[0]
	mb.CommitMsgs = mb.CommitMsgs[1:]
	return msg
}

func (node *Node) resolveMsg() {
	for {
		time.Sleep(10 * time.Microsecond)

		// Get buffered messages from the dispatcher.
		switch {
		case len(node.MsgBuffer.ReqMsgs) >= consensus.BatchSize && (node.CurrentState.LastSequenceID == -2 || node.CurrentState.CurrentStage == consensus.Committed):
			node.MsgBufferLock.ReqMsgsLock.Lock()
			// 初始化batch并确保它是非nil
			var batch consensus.BatchRequestMsg
			const viewID = 10000000000
			// 逐个赋值到数组中
			for j := 0; j < consensus.BatchSize; j++ {
				batch.Requests[j] = node.MsgBuffer.ReqMsgs[j]
			}
			batch.Timestamp = node.MsgBuffer.ReqMsgs[0].Timestamp
			batch.ClientID = node.MsgBuffer.ReqMsgs[0].ClientID
			// 添加新的批次到批次消息缓存
			node.MsgBuffer.BatchReqMsgs = append(node.MsgBuffer.BatchReqMsgs, &batch)

			errs := node.resolveRequestMsg(node.MsgBuffer.BatchReqMsgs[node.View.ID-viewID])
			if errs != nil {
				fmt.Println(errs)
				// TODO: send err to ErrorChannel
			}
			node.MsgBuffer.DequeueReqMsg()
			node.MsgBufferLock.ReqMsgsLock.Unlock()
		case len(node.MsgBuffer.PrePrepareMsgs) > 0 && (node.CurrentState.LastSequenceID == -2 || node.CurrentState.CurrentStage == consensus.Committed):
			node.MsgBufferLock.PrePrepareMsgsLock.Lock()
			errs := node.resolvePrePrepareMsg(node.MsgBuffer.PrePrepareMsgs[0])
			if errs != nil {
				fmt.Println(errs)
				// TODO: send err to ErrorChannel
			}
			node.MsgBuffer.DequeuePrePrepareMsg()
			node.MsgBufferLock.PrePrepareMsgsLock.Unlock()
		case len(node.MsgBuffer.PrepareMsgs) > 0 && node.CurrentState.CurrentStage == consensus.PrePrepared:
			node.MsgBufferLock.PrepareMsgsLock.Lock()
			var keepIndexes []int     // 用于存储需要保留的元素的索引
			var processIndex int = -1 // 用于存储第一个符合条件的元素的索引，初始化为-1表示未找到
			// 首先遍历PrepareMsgs，确定哪些元素需要保留，哪个元素需要处理
			for index, value := range node.MsgBuffer.PrepareMsgs {
				if value.ViewID < node.View.ID {
					// 不需要做任何事，因为这个元素将被删除
				} else if value.ViewID > node.View.ID {
					keepIndexes = append(keepIndexes, index) // 保留这个元素
				} else if processIndex == -1 { // 只记录第一个符合条件的元素
					processIndex = index
				} else {
					keepIndexes = append(keepIndexes, index)
				}
			}
			// 如果找到了符合条件的元素，则处理它
			if processIndex != -1 {
				errs := node.resolvePrepareMsg(node.MsgBuffer.PrepareMsgs[processIndex])
				// 将这个元素标记为已处理，不再保留
				if errs != nil {
					fmt.Println(errs)
					// TODO: send err to ErrorChannel
				}
			}
			// 创建一个新的切片来存储保留的元素
			var newPrepareMsgs []*consensus.VoteMsg // 假设YourMsgType是PrepareMsgs中元素的类型
			for _, index := range keepIndexes {
				newPrepareMsgs = append(newPrepareMsgs, node.MsgBuffer.PrepareMsgs[index])
			}

			// 更新原来的PrepareMsgs为只包含保留元素的新切片
			node.MsgBuffer.PrepareMsgs = newPrepareMsgs

			node.MsgBufferLock.PrepareMsgsLock.Unlock()

			//errs := node.resolvePrepareMsg(node.MsgBuffer.PrepareMsgs[0])
			//if errs != nil {
			//
			//	fmt.Println(errs)
			//
			//	// TODO: send err to ErrorChannel
			//}
			//node.MsgBufferLock.PrepareMsgsLock.Lock()
			//node.MsgBuffer.DequeuePrepareMsg()
			//node.MsgBufferLock.PrepareMsgsLock.Unlock()
		case len(node.MsgBuffer.CommitMsgs) > 0 && (node.CurrentState.CurrentStage == consensus.Prepared):
			node.MsgBufferLock.CommitMsgsLock.Lock()
			var keepIndexes []int // 用于存储需要保留的元素的索引
			var processIndex = -1 // 用于存储第一个符合条件的元素的索引，初始化为-1表示未找到
			// 首先遍历PrepareMsgs，确定哪些元素需要保留，哪个元素需要处理
			for index, value := range node.MsgBuffer.CommitMsgs {
				if value.ViewID < node.View.ID {
					// 不需要做任何事，因为这个元素将被删除
				} else if value.ViewID > node.View.ID {
					keepIndexes = append(keepIndexes, index) // 保留这个元素
				} else if processIndex == -1 { // 只记录第一个符合条件的元素
					processIndex = index
				} else {
					keepIndexes = append(keepIndexes, index)
				}
			}
			// 如果找到了符合条件的元素，则处理它
			if processIndex != -1 {
				errs := node.resolveCommitMsg(node.MsgBuffer.CommitMsgs[processIndex])
				// 将这个元素标记为已处理，不再保留
				if errs != nil {
					fmt.Println(errs)
					// TODO: send err to ErrorChannel
				}
			}
			// 创建一个新的切片来存储保留的元素
			var newCommitMsgs []*consensus.VoteMsg // 假设YourMsgType是PrepareMsgs中元素的类型
			for _, index := range keepIndexes {
				newCommitMsgs = append(newCommitMsgs, node.MsgBuffer.CommitMsgs[index])
			}

			// 更新原来的PrepareMsgs为只包含保留元素的新切片
			node.MsgBuffer.CommitMsgs = newCommitMsgs

			node.MsgBufferLock.CommitMsgsLock.Unlock()

		default:

		}

	}
}

func (node *Node) alarmToDispatcher() {
	for {
		time.Sleep(ResolvingTimeDuration)
		node.Alarm <- true
	}
}

func (node *Node) resolveRequestMsg(msg *consensus.BatchRequestMsg) error {

	err := node.GetReq(msg, false)
	if err != nil {
		return err
	}

	return nil
}

func (node *Node) resolvePrePrepareMsg(msg *consensus.PrePrepareMsg) error {

	// Resolve messages
	// 从下标num_of_event_to_resolve开始执行，之前执行过的PrePrepareMsg不需要再执行
	///fmt.Printf("len PrePrepareMsg msg %d\n", len(msgs))
	err := node.GetPrePrepare(msg, false)

	if err != nil {
		return err
	}

	return nil
}

func (node *Node) resolvePrepareMsg(msg *consensus.VoteMsg) error {
	// Resolve messages
	///fmt.Printf("len PrepareMsg msg %d\n", len(msgs))
	if msg.ViewID < node.View.ID {
		return nil
	}
	err := node.GetPrepare(msg)

	if err != nil {
		return err
	}

	return nil
}

func (node *Node) resolveCommitMsg(msg *consensus.VoteMsg) error {
	if msg.ViewID < node.View.ID {
		return nil
	}

	err := node.GetCommit(msg)
	if err != nil {
		return err
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
		///fmt.Println("ParsePKCS8PrivateKey err", err)
		panic(err)
	}

	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed)
	if err != nil {
		///fmt.Printf("Error from signing: %s\n", err)
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
