package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"reflect"

	"github.com/ping2h/toyrpc/pkg/dataserial"
	"github.com/ping2h/toyrpc/pkg/transport"
)

type RPCServer struct {
	addr  string
	funcs map[string]reflect.Value
}

func NewServer(addr string) *RPCServer {
	return &RPCServer{addr: addr, funcs: make(map[string]reflect.Value)}
}

func (s *RPCServer) Register(fnName string, fFunc interface{}) {
	if _, ok := s.funcs[fnName]; ok {
		return
	}
	s.funcs[fnName] = reflect.ValueOf(fFunc)
}

func (s *RPCServer) Execute(req dataserial.RPCdata) dataserial.RPCdata {
	f, ok := s.funcs[req.Name]
	if !ok {
		e := fmt.Sprintf("func %s not Registered", req.Name)
		log.Println(e)
		return dataserial.RPCdata{Name: req.Name, Args: nil, Err: e}
	}
	log.Printf("func %s is called\n", req.Name)
	inArgs := make([]reflect.Value, len(req.Args))
	for i := range req.Args {
		inArgs[i] = reflect.ValueOf(req.Args[i])
	}
	out := f.Call(inArgs)
	resArgs := make([]interface{}, len(out)-1)
	for i := 0; i < len(out)-1; i++ {
		resArgs[i] = out[i].Interface()
	}

	var er string
	if _, ok := out[len(out)-1].Interface().(error); ok {
		er = out[len(out)-1].Interface().(error).Error()
	}
	return dataserial.RPCdata{Name: req.Name, Args: resArgs, Err: er}
}

func (s *RPCServer) Run() {
	l, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.Printf("listen on %s err: %v\n", s.addr, err)
		return
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("accept err: %v\n", err)
			continue
		}
		go func() {
			connTransport := transport.NewTransport(conn)
			for {
				req, err := connTransport.Read()
				if err != nil {
					if err != io.EOF {
						log.Printf("read err: %v\n", err)
						return
					}
				}
				decReq, err := dataserial.Decode(req)
				if err != nil {
					log.Printf("error decodiing the payload err: %v\n", err)
					return
				}
				resP := s.Execute(decReq)

				b, err := dataserial.Encode(resP)
				if err != nil {
					log.Printf("error encoding the payload for response err: %v\n", err)
					return
				}
				err = connTransport.Send(b)
				if err != nil {
					log.Printf("transport write err: %v\n", err)
				}
			}
		}()
	}
}
