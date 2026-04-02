package fsthttp

import (
	"fmt"
)

// ResVPNProxyResult represents additional IP Proxy and VPN Intelligence data for a request.
type ResVPNProxyResult struct {
	IsAnonymous        bool   `json:"is_anonymous"`         // True if the IP address is present in one or more categories of anonymous flags.
	IsAnonymousVPN     bool   `json:"is_anonymous_vpn"`     // True if the IP address was identified as being from a Virtual Private Network (VPN) exit node.
	IsHostingProvider  bool   `json:"is_hosting_provider"`  // True if the IP address was identified as being from a hosting provider or data center.
	IsProxyOverVPN     bool   `json:"is_proxy_over_vpn"`    // True if the IP address was detected with the Proxy over VPN technique from premium VPN providers like ExpressVPN.
	IsPublicProxy      bool   `json:"is_public_proxy"`      // True if the IP address was identified as being from a proxy exit node.
	IsRelayProxy       bool   `json:"is_relay_proxy"`       // True if the IP address was identified as being from a relay proxy.
	IsResidentialProxy bool   `json:"is_residential_proxy"` // True if the IP address was identified as being from a proxy associated with a residential ISP.
	IsSmartDNSProxy    bool   `json:"is_smart_dns_proxy"`   // True if the IP address was identified as being from a SmartDNS exit node.
	IsTorExitNode      bool   `json:"is_tor_exit_node"`     // True if the IP address was identified as being from a Tor exit node.
	IsVPNDatacenter    bool   `json:"is_vpn_datacenter"`    // True if the IP address was identified as being part of a known VPN data center or IP address range.
	VPNServiceName     string `json:"vpn_service_name"`     // Displays the name of the VPN associated with the network of the IP address.
}

// ResVPNProxyData analyzes the current downstream request's IP address and returns VPN and proxy intelligence data.
//
// Returns an error if the ResVPNProxy feature is not enabled for your service or intelligence data is not available.
//
// Example usage:
//
//	vpnData, err := r.ResVPNProxyData()
//	if err != nil {
//	    // Feature not enabled or other error
//	    return err
//	}
func (r *Request) ResVPNProxyData() (*ResVPNProxyResult, error) {
	if r.downstream.req == nil {
		return nil, fmt.Errorf("downstream request not available")
	}

	result := &ResVPNProxyResult{}
	var err error

	result.IsAnonymous, err = r.downstream.req.DownstreamResVPNProxyIsAnonymous()
	if err = ignoreNoneError(err); err != nil {
		return nil, fmt.Errorf("is_anonymous: %w", err)
	}

	result.IsAnonymousVPN, err = r.downstream.req.DownstreamResVPNProxyIsAnonymousVPN()
	if err = ignoreNoneError(err); err != nil {
		return nil, fmt.Errorf("is_anonymous_vpn: %w", err)
	}

	result.IsHostingProvider, err = r.downstream.req.DownstreamResVPNProxyIsHostingProvider()
	if err = ignoreNoneError(err); err != nil {
		return nil, fmt.Errorf("is_hosting_provider: %w", err)
	}

	result.IsProxyOverVPN, err = r.downstream.req.DownstreamResVPNProxyIsProxyOverVPN()
	if err = ignoreNoneError(err); err != nil {
		return nil, fmt.Errorf("is_proxy_over_vpn: %w", err)
	}

	result.IsPublicProxy, err = r.downstream.req.DownstreamResVPNProxyIsPublicProxy()
	if err = ignoreNoneError(err); err != nil {
		return nil, fmt.Errorf("is_public_proxy: %w", err)
	}

	result.IsRelayProxy, err = r.downstream.req.DownstreamResVPNProxyIsRelayProxy()
	if err = ignoreNoneError(err); err != nil {
		return nil, fmt.Errorf("is_relay_proxy: %w", err)
	}

	result.IsResidentialProxy, err = r.downstream.req.DownstreamResVPNProxyIsResidentialProxy()
	if err = ignoreNoneError(err); err != nil {
		return nil, fmt.Errorf("is_residential_proxy: %w", err)
	}

	result.IsSmartDNSProxy, err = r.downstream.req.DownstreamResVPNProxyIsSmartDNSProxy()
	if err = ignoreNoneError(err); err != nil {
		return nil, fmt.Errorf("is_smart_dns_proxy: %w", err)
	}

	result.IsTorExitNode, err = r.downstream.req.DownstreamResVPNProxyIsTorExitNode()
	if err = ignoreNoneError(err); err != nil {
		return nil, fmt.Errorf("is_tor_exit_node: %w", err)
	}

	result.IsVPNDatacenter, err = r.downstream.req.DownstreamResVPNProxyIsVPNDatacenter()
	if err = ignoreNoneError(err); err != nil {
		return nil, fmt.Errorf("is_vpn_datacenter: %w", err)
	}

	result.VPNServiceName, err = r.downstream.req.DownstreamResVPNProxyVPNServiceName()
	if err = ignoreNoneError(err); err != nil {
		return nil, fmt.Errorf("vpn_service_name: %w", err)
	}

	return result, nil
}
