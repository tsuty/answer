package main

import (
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"
)

const (
	TCP = "tcp"
	UDP = "udp"
)

type Server struct {
	address      string
	readTimeout  time.Duration
	writeTimeout time.Duration
	internal     *dns.Server
	handler      *Handler
}

type Servers struct {
	tcp *Server
	udp *Server
}

func NewServers(host, port, rt, wt string, ns []string) (*Servers, error) {
	tcp, err := NewNameServer(host, port, rt, wt, ns)
	if err != nil {
		return nil, err
	}
	tcp.SetupTCPServer()

	udp, err := NewNameServer(host, port, rt, wt, ns)
	if err != nil {
		return nil, err
	}
	udp.SetupUDPServer()

	return &Servers{
		tcp: tcp,
		udp: udp,
	}, nil
}

func (ss Servers) Start() (err error) {
	defer func() {
		if e := recover(); e != nil {
			if er, ok := e.(error); ok {
				err = er
			} else {
				err = fmt.Errorf("panic recover %+v", e)
			}
		}
	}()

	go ss.tcp.Start()
	go ss.udp.Start()

	return nil
}

func (s *Server) SetupTCPServer() {
	handler := dns.NewServeMux()
	handler.HandleFunc(".", s.handler.TCP)

	s.internal = &dns.Server{
		Addr:         s.address,
		Net:          TCP,
		Handler:      handler,
		ReadTimeout:  s.readTimeout,
		WriteTimeout: s.writeTimeout,
	}

	s.handler.client = &dns.Client{
		Net:          TCP,
		ReadTimeout:  s.readTimeout,
		WriteTimeout: s.writeTimeout,
	}
}

func (s *Server) SetupUDPServer() {
	handler := dns.NewServeMux()
	handler.HandleFunc(".", s.handler.UDP)

	s.internal = &dns.Server{
		Addr:         s.address,
		Net:          UDP,
		Handler:      handler,
		UDPSize:      65535,
		ReadTimeout:  s.readTimeout,
		WriteTimeout: s.writeTimeout,
	}

	s.handler.client = &dns.Client{
		Net:          UDP,
		ReadTimeout:  s.readTimeout,
		WriteTimeout: s.writeTimeout,
	}
}

func NewNameServer(host, port, rt, wt string, ns []string) (*Server, error) {
	address := net.JoinHostPort(host, port)

	rtd, err := time.ParseDuration(rt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse duration %s %w", rt, err)
	}
	wtd, err := time.ParseDuration(wt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse duration %s %w", rt, err)
	}

	return &Server{
		address:      address,
		readTimeout:  rtd,
		writeTimeout: wtd,
		handler:      NewHandler(ns),
	}, nil
}

func (s *Server) Start() {
	if s.internal == nil {
		msg := "can not start server. server not initialized"
		logger.Error(msg)
		panic(msg)
	}

	logger.Info("listen and serve %s", s.internal.Net)
	if err := s.internal.ListenAndServe(); err != nil {
		msg := fmt.Sprintf("can not start server. %s server (%s)", s.internal.Net, err.Error())
		logger.Error(msg)
		panic(msg)
	}
}
