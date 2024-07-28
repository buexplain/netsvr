// Code generated by 'yaegi extract net/netip'. DO NOT EDIT.

//go:build go1.21 && !go1.22
// +build go1.21,!go1.22

package stdlib

import (
	"net/netip"
	"reflect"
)

func init() {
	Symbols["net/netip/netip"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"AddrFrom16":              reflect.ValueOf(netip.AddrFrom16),
		"AddrFrom4":               reflect.ValueOf(netip.AddrFrom4),
		"AddrFromSlice":           reflect.ValueOf(netip.AddrFromSlice),
		"AddrPortFrom":            reflect.ValueOf(netip.AddrPortFrom),
		"IPv4Unspecified":         reflect.ValueOf(netip.IPv4Unspecified),
		"IPv6LinkLocalAllNodes":   reflect.ValueOf(netip.IPv6LinkLocalAllNodes),
		"IPv6LinkLocalAllRouters": reflect.ValueOf(netip.IPv6LinkLocalAllRouters),
		"IPv6Loopback":            reflect.ValueOf(netip.IPv6Loopback),
		"IPv6Unspecified":         reflect.ValueOf(netip.IPv6Unspecified),
		"MustParseAddr":           reflect.ValueOf(netip.MustParseAddr),
		"MustParseAddrPort":       reflect.ValueOf(netip.MustParseAddrPort),
		"MustParsePrefix":         reflect.ValueOf(netip.MustParsePrefix),
		"ParseAddr":               reflect.ValueOf(netip.ParseAddr),
		"ParseAddrPort":           reflect.ValueOf(netip.ParseAddrPort),
		"ParsePrefix":             reflect.ValueOf(netip.ParsePrefix),
		"PrefixFrom":              reflect.ValueOf(netip.PrefixFrom),

		// type definitions
		"Addr":     reflect.ValueOf((*netip.Addr)(nil)),
		"AddrPort": reflect.ValueOf((*netip.AddrPort)(nil)),
		"Prefix":   reflect.ValueOf((*netip.Prefix)(nil)),
	}
}
