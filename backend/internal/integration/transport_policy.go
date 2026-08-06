package integration

import (
	"context"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

// HostResolver 是可替换的 DNS 解析端口。生产默认使用系统解析器，测试可注入固定结果。
type HostResolver interface {
	LookupIP(context.Context, string) ([]net.IP, error)
}

type systemHostResolver struct{}

func (systemHostResolver) LookupIP(ctx context.Context, host string) ([]net.IP, error) {
	return net.DefaultResolver.LookupIP(ctx, "ip", host)
}

// EndpointPolicy 冻结 HTTP 协议与网络地址边界。AllowHTTP 只能由服务端配置显式开启。
type EndpointPolicy struct {
	allowHTTP               bool
	approvedPrivatePrefixes []netip.Prefix
	resolver                HostResolver
}

// NewEndpointPolicy 创建端点安全策略。空 resolver 使用系统 DNS；私网前缀仅用于明确批准的内部服务。
func NewEndpointPolicy(allowHTTP bool, approvedPrivateCIDRs []string, resolver HostResolver) (EndpointPolicy, error) {
	prefixes := make([]netip.Prefix, 0, len(approvedPrivateCIDRs))
	for _, raw := range approvedPrivateCIDRs {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(raw))
		if err != nil {
			return EndpointPolicy{}, newTransportError(TransportErrorInvalidConfig)
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	if resolver == nil {
		resolver = systemHostResolver{}
	}
	return EndpointPolicy{allowHTTP: allowHTTP, approvedPrivatePrefixes: prefixes, resolver: resolver}, nil
}

// DefaultEndpointPolicy 默认只允许 HTTPS 和可公开访问的地址。
func DefaultEndpointPolicy() EndpointPolicy {
	policy, _ := NewEndpointPolicy(false, nil, nil)
	return policy
}

func (p EndpointPolicy) validateTarget(ctx context.Context, target *url.URL) ([]net.IP, error) {
	if target == nil || target.Hostname() == "" {
		return nil, newTransportError(TransportErrorInvalidURL)
	}
	if target.Scheme != "https" && (target.Scheme != "http" || !p.allowHTTP) {
		return nil, newTransportError(TransportErrorSSRFRejected)
	}
	host := strings.TrimSuffix(strings.ToLower(target.Hostname()), ".")
	if host == "" {
		return nil, newTransportError(TransportErrorInvalidURL)
	}
	if literal := net.ParseIP(host); literal != nil {
		if err := p.validateIP(literal); err != nil {
			return nil, err
		}
		return []net.IP{literal}, nil
	}
	resolver := p.resolver
	if resolver == nil {
		resolver = systemHostResolver{}
	}
	addresses, err := resolver.LookupIP(ctx, host)
	if err != nil || len(addresses) == 0 {
		return nil, newTransportError(TransportErrorNetwork)
	}
	for _, address := range addresses {
		if err := p.validateIP(address); err != nil {
			return nil, err
		}
	}
	return addresses, nil
}

func (p EndpointPolicy) validateIP(value net.IP) error {
	address, ok := netip.AddrFromSlice(value)
	if !ok {
		return newTransportError(TransportErrorSSRFRejected)
	}
	address = address.Unmap()
	if address.IsUnspecified() || address.IsLoopback() || address.IsLinkLocalUnicast() ||
		address.IsLinkLocalMulticast() || address.IsInterfaceLocalMulticast() || address.IsMulticast() ||
		isCloudMetadataAddress(address) {
		return newTransportError(TransportErrorSSRFRejected)
	}
	if address.IsPrivate() || isCarrierGradeNAT(address) {
		for _, prefix := range p.approvedPrivatePrefixes {
			if prefix.Contains(address) {
				return nil
			}
		}
		return newTransportError(TransportErrorSSRFRejected)
	}
	return nil
}

func isCarrierGradeNAT(address netip.Addr) bool {
	carrierGrade, _ := netip.ParsePrefix("100.64.0.0/10")
	return carrierGrade.Contains(address)
}

func isCloudMetadataAddress(address netip.Addr) bool {
	for _, raw := range []string{"169.254.169.254", "100.100.100.200", "fd00:ec2::254"} {
		metadata, err := netip.ParseAddr(raw)
		if err == nil && metadata == address {
			return true
		}
	}
	return false
}
