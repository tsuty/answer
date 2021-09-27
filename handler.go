package main

import (
	"net"

	"github.com/miekg/dns"
)

type Handler struct {
	client    *dns.Client
	addresses []string
}

func NewHandler(ns []string) *Handler {
	h := &Handler{}
	for _, n := range ns {
		if _, _, err := net.SplitHostPort(n); err != nil {
			n = net.JoinHostPort(n, "53")
		}
		h.addresses = append(h.addresses, n)
	}
	return h
}

func (h *Handler) TCP(w dns.ResponseWriter, req *dns.Msg) {
	addr := w.RemoteAddr().(*net.TCPAddr)
	logger.Info("%s (TCP) handle", addr)
	logger.Debug("request\n%s", req)
	res := h.HandleRequest(req)
	if len(res.Answer) == 0 && len(h.addresses) > 0 {
		logger.Debug("no answer")
		res = h.Exchange(req)
	}

	for _, a := range res.Answer {
		logger.Info("answer %s", a.String())
	}

	w.WriteMsg(res)
}

func (h *Handler) UDP(w dns.ResponseWriter, req *dns.Msg) {
	addr := w.RemoteAddr().(*net.UDPAddr)
	logger.Info("%s (UDP) handle", addr)
	logger.Debug("request\n%s", req)
	res := h.HandleRequest(req)
	if len(res.Answer) == 0 && len(h.addresses) > 0 {
		logger.Debug("no answer")
		res = h.Exchange(req.SetEdns0(65535, true))
	}

	for _, a := range res.Answer {
		logger.Info("answer %s", a.String())
	}

	w.WriteMsg(res)
}

func (h *Handler) HandleRequest(req *dns.Msg) *dns.Msg {
	if len(req.Question) == 0 {
		return ServerFailure(req)
	}

	question := req.Question[0]
	logger.Info("question %s", question.String())

	switch question.Qclass {
	case dns.ClassINET:
	default:
		return NotImplement(req)
	}

	header := dns.RR_Header{
		Name:   question.Name,
		Rrtype: question.Qtype,
		Class:  question.Qclass,
	}
	name := question.Name
	if dns.IsFqdn(name) {
		name = name[:len(name)-1]
	}

	m := new(dns.Msg)
	m.SetReply(req)

	switch question.Qtype {
	case dns.TypeA,
		dns.TypeAAAA:
		ips, err := net.LookupIP(name)
		if err != nil {
			logger.Notice("can not lookup ip %s", err.Error())
			return m
		}

		for _, ip := range ips {
			if header.Rrtype == dns.TypeA {
				m.Answer = append(m.Answer, &dns.A{
					Hdr: header,
					A:   ip,
				})
			} else {
				m.Answer = append(m.Answer, &dns.AAAA{
					Hdr:  header,
					AAAA: ip,
				})
			}
		}
	case dns.TypeCNAME:
		cname, err := net.LookupCNAME(name)
		if err != nil {
			logger.Notice("can not lookup cname %s", err.Error())
			return m
		}

		m.Answer = append(m.Answer, &dns.CNAME{
			Hdr:    header,
			Target: cname,
		})
	case dns.TypeNS:
		nss, err := net.LookupNS(name)
		if err != nil {
			logger.Notice("can not lookup ns %s", err.Error())
			return m
		}

		for _, ns := range nss {
			m.Answer = append(m.Answer, &dns.NS{
				Hdr: header,
				Ns:  ns.Host,
			})
		}
	case dns.TypeMX:
		mxs, err := net.LookupMX(name)
		if err != nil {
			logger.Notice("can not lookup mx %s", err.Error())
			return m
		}

		for _, mx := range mxs {
			m.Answer = append(m.Answer, &dns.MX{
				Hdr:        header,
				Preference: mx.Pref,
				Mx:         mx.Host,
			})
		}
	case dns.TypeTXT:
		txt, err := net.LookupTXT(name)
		if err != nil {
			logger.Notice("can not lookup txt %s", err.Error())
			return m
		}

		m.Answer = append(m.Answer, &dns.TXT{
			Hdr: header,
			Txt: txt,
		})
	case dns.TypePTR:
		m := new(dns.Msg)
		m.SetReply(req)

		names, err := net.LookupAddr(name)
		if err != nil {
			logger.Notice("can not lookup ptr %s", err.Error())
			return m
		}

		for _, name := range names {
			m.Answer = append(m.Answer, &dns.PTR{
				Hdr: header,
				Ptr: name,
			})
		}
	}

	return m
}

func (h *Handler) Exchange(req *dns.Msg) *dns.Msg {
	for _, address := range h.addresses {
		r, rtt, err := h.client.Exchange(req, address)
		if err != nil {
			logger.Warn("socket error on %s %v", address, err)
			continue
		}

		if r != nil && r.Rcode != dns.RcodeSuccess {
			logger.Warn("failed to get an valid answer on %s", address)
			continue
		}

		logger.Info("resolve on %s rtt: %v", address, rtt)
		return r
	}

	m := new(dns.Msg)
	m.SetReply(req)

	return m
}

func NotImplement(req *dns.Msg) *dns.Msg {
	m := new(dns.Msg)
	m.SetRcode(req, dns.RcodeNotImplemented)
	return m
}

func ServerFailure(req *dns.Msg) *dns.Msg {
	m := new(dns.Msg)
	m.SetRcode(req, dns.RcodeServerFailure)
	return m
}
