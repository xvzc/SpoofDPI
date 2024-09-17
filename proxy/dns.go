package proxy

import (
	"context"
	"net"

	"github.com/xvzc/SpoofDPI/dns/resolver"
	dnshandler "github.com/xvzc/SpoofDPI/dns/resolver/handler"
	"github.com/xvzc/SpoofDPI/util"
	"github.com/xvzc/SpoofDPI/util/log"
)

type proxyDnsFactory struct {
	systemClient       *resolver.SystemResolver
	generalClient      *resolver.GeneralResolver
	dohClient          *resolver.DOHResolver
	shouldUseSystemDns func(host string) bool
	enableDOH          bool
}

func newProxyDnsResolver(dnsAddr string, dnsPort string, enableDOH bool, shouldUseSystemResolver func(host string) bool) resolver.Resolver {
	factoryResolver := resolver.NewFactoryResolver(
		newProxyDnsFactory(
			dnsAddr,
			dnsPort,
			shouldUseSystemResolver,
			enableDOH,
		),
	)
	return dnshandler.Apply(factoryResolver, dnshandler.NewLoggingHandler(), dnshandler.NewErrorHandler())
}

func newProxyDnsFactory(
	dnsAddr string,
	dnsPort string,
	shouldUseSystemDnsFunc func(host string) bool,
	enableDOH bool) *proxyDnsFactory {
	return &proxyDnsFactory{
		systemClient:       resolver.NewSystemResolver(),
		dohClient:          resolver.NewDOHResolver(dnsAddr),
		generalClient:      resolver.NewGeneralResolver(net.JoinHostPort(dnsAddr, dnsPort)),
		enableDOH:          enableDOH,
		shouldUseSystemDns: shouldUseSystemDnsFunc,
	}
}

func (p *proxyDnsFactory) Get(ctx context.Context, host string, _ []uint16) (resolver.Resolver, error) {
	l := util.GetCtxWithScope(ctx, scopeProxy)
	logger := log.GetCtxLogger(l)
	r, _ := p.findInternal(host)
	logger.Debug().Msgf("proxy dns factory returns: %v", r)
	return r, nil
}

func (p *proxyDnsFactory) findInternal(host string) (resolver.Resolver, error) {
	if p.shouldUseSystemDns(host) {
		return p.systemClient, nil
	}
	if p.enableDOH {
		return p.dohClient, nil
	}
	return p.generalClient, nil
}
