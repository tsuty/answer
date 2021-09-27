package main

import (
	"net"

	"github.com/miekg/dns"
)

type Handler struct{}

func (h *Handler) TCP(w dns.ResponseWriter, req *dns.Msg) {
	addr := w.RemoteAddr().(*net.TCPAddr)
	logger.Info("%s (TCP) handle", addr)
	logger.Debug("request\n%s", req)
	w.WriteMsg(h.HandleRequest(req))
}

func (h *Handler) UDP(w dns.ResponseWriter, req *dns.Msg) {
	addr := w.RemoteAddr().(*net.UDPAddr)
	logger.Info("%s (UDP) handle", addr)
	logger.Debug("request\n%s", req)
	w.WriteMsg(h.HandleRequest(req))
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

	switch question.Qtype {
	case dns.TypeA,
		dns.TypeAAAA:
		m := new(dns.Msg)
		m.SetReply(req)

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

		return m
	case dns.TypeCNAME:
		m := new(dns.Msg)
		m.SetReply(req)

		cname, err := net.LookupCNAME(name)
		if err != nil {
			logger.Notice("can not lookup cname %s", err.Error())
			return m
		}

		m.Answer = append(m.Answer, &dns.CNAME{
			Hdr:    header,
			Target: cname,
		})

		return m
	case dns.TypeNS:
		m := new(dns.Msg)
		m.SetReply(req)

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

		return m
	case dns.TypeMX:
		m := new(dns.Msg)
		m.SetReply(req)

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

		return m
	case dns.TypeTXT:
		m := new(dns.Msg)
		m.SetReply(req)

		txt, err := net.LookupTXT(name)
		if err != nil {
			logger.Notice("can not lookup txt %s", err.Error())
			return m
		}

		m.Answer = append(m.Answer, &dns.TXT{
			Hdr: header,
			Txt: txt,
		})

		return m
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

		return m
	default:

		return NotImplement(req)
	}
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
