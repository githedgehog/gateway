# NAT Design brainstorm for 26.01

This is a sample YAML configuration for NAT with stateful NAT, stateless NAT, port forwarding and masquerading.

## For the existing API

I think we agree to this section.

Drop `stateless` and `stateful` inside the `nat` section.
Add `portForward`, `masquerade` and `static` options.
`static` is the default nat type

## Changes to what is in master

For now we'll remove `ports` from the `ips` and `as` block so that there is no way to configure stateless port forwarding.
But we'll keep the implementation so we can expose it again later with different configuration.

### Option
Allow a port mapping configuration inside the `static` NAT section.  

> Note(MV): I oppose this option because we would then have to remove this config later, potentially leading to confusion and maintenance issues. I suggest we fallback to this option at the last minute if needed.

```yaml
  peerings:
    # I want my ssh server at 192.168.1.1:22  to be reachable at  77.77.77.77:1001 from VPC external
    # I want my ssh server at 192.168.1.2:22  to be reachable at  77.77.77.77:1002    "  "    "
    # 
    # For port fowarding this approach is the winner!
    vpc-1-external:
      gatewayGroup: gw-group-1
      peering:
        VPC-1:
          expose:
            - ips:
                - cidr: 192.168.1.0/24
              as:
                - cidr: 77.77.77.77/31 # this does not even work
              nat:
                masquerade: { idleTimout: "3m" }
            - ips:
                - cidr: 192.168.1.1/32
              as:
                - cidr: 77.77.77.77/32
              nat:
                portForward:
                  ports:
                    - proto: tcp
                      port: 22-23
                      as: 1122-1123
                    - proto: udp
                      port: "23"
                      as: "1123"
            - ips:
                - cidr: 192.168.1.2/32
              as:
                - cidr: 77.77.77.77/32
              nat:
                portForward: 
                  ports:
                    - proto: tcp
                      port: 22
                      as: 1222

        VPC-external:
          expose:
            - default: true

    # No masquerading only port forwarding
    # I want my ssh server at 192.168.1.1:22  to be reachable at  77.77.77.77:1003
    vpc-2-external:
      gatewayGroup: gw-group-1
      peering:
        VPC-2:
          expose:
            - ips:
                - cidr: 192.168.1.1/32
              as:
                - cidr: 77.77.77.77/32
              nat:
                portForward: 
                  ports:
                    - proto: tcp
                      port: 22
                      as: 2122
        VPC-external:
          expose:
            - default: true

    # With masquerade
    # I want my ssh server at 192.168.1.1:22  to be reachable at  77.77.77.77:1004
    vpc-3-external:
      gatewayGroup: gw-group-1
      peering:
        VPC-3:
          expose:
            - ips:
                - cidr: 192.168.1.0/24
              as:
                - cidr: 77.77.77.77/32 # this does not even work
              nat:
                masquerade: {}
            - ips:
                - cidr: 192.168.1.1/31
                  port: 22-24
                - cidr: 192.168.1.2/32
                  port: 22-24
              as:
                - cidr: 77.77.77.77/32
                  port: 3122-3141
              nat:
                portForward: {}
            - ips:
                - cidr: 192.168.1.2/32
                  port: 23
              as:
                - cidr: 77.77.77.78/32
                  port: 3123
              nat:
                portForward: 
                  protocol: udp

        VPC-external:
          expose:
            - default: true
```
