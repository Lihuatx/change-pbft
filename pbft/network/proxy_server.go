package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"simple_pbft/pbft/consensus"
	"time"
)

type Server struct {
	url  string
	node *Node
}

func NewServer(nodeID string) *Server {
	node := NewNode(nodeID)
	server := &Server{node.NodeTable[nodeID], node}

	server.setRoute()

	return server
}

func (server *Server) Start() {
	fmt.Printf("Server will be started at %s...\n", server.url)
	if err := http.ListenAndServe(server.url, nil); err != nil {
		fmt.Println(err)
		return
	}
}

func (server *Server) setRoute() {
	http.HandleFunc("/req", server.getReq)
	http.HandleFunc("/preprepare", server.getPrePrepare)
	http.HandleFunc("/prepare", server.getPrepare)
	http.HandleFunc("/commit", server.getCommit)
	http.HandleFunc("/reply", server.getReply)
}

func (server *Server) getReq(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.RequestMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	// for test
	if err != nil {
		fmt.Println(err)
		return
	}
	// 保存请求的URL到RequestMsg中
	// 获取客户端地址

	server.node.MsgEntrance <- &msg
	//server.node.MsgBufferLock.ReqMsgsLock.Lock()
	//server.node.MsgBuffer.ReqMsgs = append(server.node.MsgBuffer.ReqMsgs, msg)
	//server.node.MsgBufferLock.ReqMsgsLock.Unlock()
	time.Sleep(100 * time.Millisecond) // 程序暂停50毫秒
}

func (server *Server) getPrePrepare(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.PrePrepareMsg
	err := json.NewDecoder(request.Body).Decode(&msg)

	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Got PrePrepare %d", msg.ViewID, msg.NodeID, msg.SequenceID)
	server.node.MsgEntrance <- &msg
	//server.node.MsgBufferLock.PrePrepareMsgsLock.Lock()
	//server.node.MsgBuffer.PrePrepareMsgs = append(server.node.MsgBuffer.PrePrepareMsgs, msg)
	//server.node.MsgBufferLock.PrePrepareMsgsLock.Unlock()
	//time.Sleep(50 * time.Millisecond) // 程序暂停50毫秒

}

func (server *Server) getPrepare(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.VoteMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}

	server.node.MsgEntrance <- &msg
	//server.node.MsgBufferLock.PrepareMsgsLock.Lock()
	//server.node.MsgBuffer.PrepareMsgs = append(server.node.MsgBuffer.PrepareMsgs, msg)
	//server.node.MsgBufferLock.PrepareMsgsLock.Unlock()
	//time.Sleep(50 * time.Millisecond) // 程序暂停50毫秒

}

func (server *Server) getCommit(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.VoteMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}

	server.node.MsgEntrance <- &msg
	//server.node.MsgBufferLock.CommitMsgsLock.Lock()
	//server.node.MsgBuffer.CommitMsgs = append(server.node.MsgBuffer.CommitMsgs, msg)
	//server.node.MsgBufferLock.CommitMsgsLock.Unlock()
	//time.Sleep(50 * time.Millisecond) // 程序暂停50毫秒

}

func (server *Server) getReply(writer http.ResponseWriter, request *http.Request) {
	var msg consensus.ReplyMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		fmt.Println(err)
		return
	}

	server.node.GetReply(&msg)
}

func send(url string, msg []byte) {
	buff := bytes.NewBuffer(msg)
	http.Post("http://"+url, "application/json", buff)
}
