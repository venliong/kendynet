package rpc

import (
	"sync"
	"fmt"
	"runtime"
	"github.com/sniperHW/kendynet/util"
)

type RPCReplyer struct {
	encoder RPCMessageEncoder
	channel RPCChannel
	req    *RPCRequest			
}

func (this *RPCReplyer) Reply(ret interface{},err error) {
	if this.req.NeedResp {
		response := &RPCResponse{Seq:this.req.Seq,Ret:ret}
		this.reply(response)
	}
}

func (this *RPCReplyer) reply(response RPCMessage) {
	msg,err := this.encoder.Encode(response)
	if nil != err {
		Logger.Errorf(util.FormatFileLine("Encode rpc response error:%s\n",err.Error()))
		return
	}
	err = this.channel.SendRPCResponse(msg)
	if nil != err {		
		Logger.Errorf(util.FormatFileLine("send rpc response to (%s) error:%s\n",this.channel.Name() , err.Error()))
	}	
}

type RPCMethodHandler func (*RPCReplyer,interface{})

type RPCServer struct {
	encoder   		 RPCMessageEncoder
	decoder   		 RPCMessageDecoder
	methods   		 map[string]RPCMethodHandler
	mutexMethods     sync.Mutex
}

func (this *RPCServer) RegisterMethod(name string,method RPCMethodHandler) error {
	if name == "" {
		return fmt.Errorf("name == ''")
	}

	if nil == method {
		return fmt.Errorf("method == nil")		
	}

	defer func(){
		this.mutexMethods.Unlock()
	}()
	this.mutexMethods.Lock()

	_,ok := this.methods[name]
	if ok {
		return fmt.Errorf("duplicate method:%s",name)
	} 
	this.methods[name] = method
	return nil
}

func (this *RPCServer) UnRegisterMethod(name string) {
	defer func(){
		this.mutexMethods.Unlock()
	}()
	this.mutexMethods.Lock()
	delete(this.methods,name)	
}

func (this *RPCServer) callMethod(method RPCMethodHandler,replyer *RPCReplyer,arg interface{}) (err error) {
	defer func(){
		if r := recover(); r != nil {
			buf := make([]byte, 65535)
			l := runtime.Stack(buf, false)
			err = fmt.Errorf("%v: %s", r, buf[:l])
			Logger.Errorf(util.FormatFileLine("%s\n",err.Error()))
		}			
	}()
	method(replyer,arg)
	return
}

func (this *RPCServer) OnRPCMessage(channel RPCChannel,message interface{}) {
	msg,err := this.decoder.Decode(message)
	if nil != err {
		Logger.Errorf(util.FormatFileLine("RPCServer rpc message from(%s) decode err:%s\n",channel.Name,err.Error()))
		return
	}

	switch msg.(type) {
		case *Ping:{
			ping := msg.(*Ping)
			pong := &Pong{Seq:ping.Seq,TimeStamp:ping.TimeStamp}
			replyer := &RPCReplyer{encoder:this.encoder,channel:channel}
			replyer.reply(pong)
			return			
		}
		case *RPCRequest: {
			req := msg.(*RPCRequest)
			this.mutexMethods.Lock()
			method,ok := this.methods[req.Method]
			this.mutexMethods.Unlock()
			if !ok {
				err = fmt.Errorf("invaild method:%s",req.Method)
				Logger.Errorf(util.FormatFileLine("rpc request from(%s) invaild method %s\n",channel.Name(),req.Method))		
			}

			replyer := &RPCReplyer{encoder:this.encoder,channel:channel,req:req}
			if nil == err {
				err = this.callMethod(method,replyer,req.Arg)
			}

			if nil != err && req.NeedResp {
				response := &RPCResponse{Seq:req.Seq,Err:err}
				replyer.reply(response)
			}
			return			
		}
		default: {

		}
	}

}

func NewRPCServer(decoder  RPCMessageDecoder,encoder RPCMessageEncoder) (*RPCServer,error) {
	if nil == decoder {
		return nil,fmt.Errorf("decoder == nil")
	}

	if nil == encoder {
		return nil,fmt.Errorf("encoder == nil")
	}

	mgr := &RPCServer{decoder:decoder,encoder:encoder}
	mgr.methods = make(map[string]RPCMethodHandler)
	return mgr,nil
}