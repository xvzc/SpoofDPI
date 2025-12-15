package config

import (
	"fmt"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/xvzc/SpoofDPI/internal/ptr"
)

// ┌─────────────────┐
// │ GENERAL OPTIONS │
// └─────────────────┘
var _ merger[*GeneralOptions] = (*GeneralOptions)(nil)

var availableLogLevels = []string{"info", "warn", "trace", "error", "debug"}

type GeneralOptions struct {
	LogLevel       *zerolog.Level `toml:"log-level"`
	Silent         *bool          `toml:"silent"`
	SetSystemProxy *bool          `toml:"system-proxy"`
}

func (o *GeneralOptions) UnmarshalTOML(data any) (err error) {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("non-table type general config")
	}

	o.Silent = findFrom(m, "silent", parseBoolFn(), &err)
	o.SetSystemProxy = findFrom(m, "system-proxy", parseBoolFn(), &err)
	if p := findFrom(m, "log-level", parseStringFn(checkLogLevel), &err); isOk(p, err) {
		o.LogLevel = ptr.FromValue(MustParseLogLevel(*p))
	}

	return err
}

func (o *GeneralOptions) Clone() *GeneralOptions {
	if o == nil {
		return nil
	}

	var newLevel *zerolog.Level
	if o.LogLevel != nil {
		newLevel = ptr.FromValue(MustParseLogLevel(strings.ToLower(o.LogLevel.String())))
	}

	return &GeneralOptions{
		LogLevel:       newLevel,
		Silent:         ptr.Clone(o.Silent),
		SetSystemProxy: ptr.Clone(o.SetSystemProxy),
	}
}

func (origin *GeneralOptions) Merge(overrides *GeneralOptions) *GeneralOptions {
	if overrides == nil {
		return origin.Clone()
	}

	if origin == nil {
		return overrides.Clone()
	}

	return &GeneralOptions{
		LogLevel:       ptr.CloneOr(overrides.LogLevel, origin.LogLevel),
		Silent:         ptr.CloneOr(overrides.Silent, origin.Silent),
		SetSystemProxy: ptr.CloneOr(overrides.SetSystemProxy, origin.SetSystemProxy),
	}
}

// ┌────────────────┐
// │ SERVER OPTIONS │
// └────────────────┘
var _ merger[*ServerOptions] = (*ServerOptions)(nil)

type ServerOptions struct {
	DefaultTTL *uint8         `toml:"default-ttl"`
	ListenAddr *net.TCPAddr   `toml:"listen-addr"`
	Timeout    *time.Duration `toml:"timeout"`
}

func (o *ServerOptions) UnmarshalTOML(data any) (err error) {
	v, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("non-table type server config")
	}

	o.DefaultTTL = findFrom(v, "default-ttl", parseIntFn[uint8](checkUint8NonZero), &err)

	if p := findFrom(v, "listen-addr", parseStringFn(checkHostPort), &err); isOk(p, err) {
		o.ListenAddr = ptr.FromValue(MustParseTCPAddr(*p))
	}

	if p := findFrom(v, "timeout", parseIntFn[uint16](checkUint16), &err); isOk(p, err) {
		o.Timeout = ptr.FromValue(time.Duration(*p) * time.Millisecond)
	}

	return err
}

func (o *ServerOptions) Clone() *ServerOptions {
	if o == nil {
		return nil
	}

	var newAddr *net.TCPAddr
	if o.ListenAddr != nil {
		newAddr = &net.TCPAddr{
			IP:   append(net.IP(nil), o.ListenAddr.IP...),
			Port: o.ListenAddr.Port,
			Zone: o.ListenAddr.Zone,
		}
	}

	return &ServerOptions{
		DefaultTTL: ptr.Clone(o.DefaultTTL),
		ListenAddr: newAddr,
		Timeout:    ptr.Clone(o.Timeout),
	}
}

func (origin *ServerOptions) Merge(overrides *ServerOptions) *ServerOptions {
	if overrides == nil {
		return origin.Clone()
	}

	if origin == nil {
		return overrides.Clone()
	}

	return &ServerOptions{
		DefaultTTL: ptr.CloneOr(overrides.DefaultTTL, origin.DefaultTTL),
		ListenAddr: ptr.CloneOr(overrides.ListenAddr, origin.ListenAddr),
		Timeout:    ptr.CloneOr(overrides.Timeout, origin.Timeout),
	}
}

// ┌─────────────┐
// │ DNS OPTIONS │
// └─────────────┘
var _ merger[*DNSOptions] = (*DNSOptions)(nil)

type (
	DNSModeType  int
	DNSQueryType int
)

var (
	availableDNSModes   = []string{"udp", "https", "system"}
	availableDNSQueries = []string{"ipv4", "ipv6", "all"}
)

const (
	DNSModeUDP DNSModeType = iota
	DNSModeHTTPS
	DNSModeSystem
)

const (
	DNSQueryIPv4 DNSQueryType = iota
	DNSQueryIPv6
	DNSQueryAll
)

func (t DNSModeType) String() string {
	return availableDNSModes[t]
}

func (t DNSQueryType) String() string {
	return availableDNSQueries[t]
}

type DNSOptions struct {
	Mode     *DNSModeType  `toml:"mode"      json:"mo,omitempty"`
	Addr     *net.TCPAddr  `toml:"addr"      json:"ad,omitempty"`
	HTTPSURL *string       `toml:"https-url" json:"hu,omitempty"`
	QType    *DNSQueryType `toml:"qtype"     json:"qt:omitempty"`
	Cache    *bool         `toml:"cache"     json:"ca:omitempty"`
}

func (o *DNSOptions) UnmarshalTOML(data any) (err error) {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("'dns' must be table type")
	}

	if p := findFrom(m, "mode", parseStringFn(checkDNSMode), &err); isOk(p, err) {
		o.Mode = ptr.FromValue(MustParseDNSModeType(*p))
	}

	if p := findFrom(m, "addr", parseStringFn(checkHostPort), &err); isOk(p, err) {
		o.Addr = ptr.FromValue(MustParseTCPAddr(*p))
	}

	o.HTTPSURL = findFrom(m, "https-url", parseStringFn(checkHTTPSEndpoint), &err)

	if p := findFrom(m, "qtype", parseStringFn(checkDNSQueryType), &err); isOk(p, err) {
		o.QType = ptr.FromValue(MustParseDNSQueryType(*p))
	}

	o.Cache = findFrom(m, "cache", parseBoolFn(), &err)

	return
}

func (o *DNSOptions) Clone() *DNSOptions {
	if o == nil {
		return nil
	}

	var newAddr *net.TCPAddr
	if o.Addr != nil {
		newAddr = &net.TCPAddr{
			IP:   append(net.IP(nil), o.Addr.IP...),
			Port: o.Addr.Port,
			Zone: o.Addr.Zone,
		}
	}

	return &DNSOptions{
		Mode:     ptr.Clone(o.Mode),
		Addr:     newAddr,
		HTTPSURL: ptr.Clone(o.HTTPSURL),
		QType:    ptr.Clone(o.QType),
		Cache:    ptr.Clone(o.Cache),
	}
}

func (origin *DNSOptions) Merge(overrides *DNSOptions) *DNSOptions {
	if overrides == nil {
		return origin.Clone()
	}

	if origin == nil {
		return overrides.Clone()
	}

	return &DNSOptions{
		Mode:     ptr.CloneOr(overrides.Mode, origin.Mode),
		Addr:     ptr.CloneOr(overrides.Addr, origin.Addr),
		HTTPSURL: ptr.CloneOr(overrides.HTTPSURL, origin.HTTPSURL),
		QType:    ptr.CloneOr(overrides.QType, origin.QType),
		Cache:    ptr.CloneOr(overrides.Cache, origin.Cache),
	}
}

// ┌───────────────┐
// │ HTTPS OPTIONS │
// └───────────────┘
var _ merger[*HTTPSOptions] = (*HTTPSOptions)(nil)

const FakeClientHello = "" +
	"\x16\x03\x01\x02\x00\x01\x00\x01\xfc\x03\x03\x9a\x8f\xa7" +
	"\x6a\x5d\x57\xf3\x62\x19\xbe\x46\x82\x45\xe2\x59\x5c\xb4" +
	"\x48\x31\x12\x15\x14\x79\x2c\xaa\xcd\xea\xda\xf0\xe1\xfd" +
	"\xbb\x20\xf4\x83\x2a\x94\xf1\x48\x3b\x9d\xb6\x74\xba\x3c" +
	"\x81\x63\xbc\x18\xcc\x14\x45\x57\x6c\x80\xf9\x25\xcf\x9c" +
	"\x86\x60\x50\x31\x2e\xe9\x00\x22\x13\x01\x13\x03\x13\x02" +
	"\xc0\x2b\xc0\x2f\xcc\xa9\xcc\xa8\xc0\x2c\xc0\x30\xc0\x0a" +
	"\xc0\x09\xc0\x13\xc0\x14\x00\x33\x00\x39\x00\x2f\x00\x35" +
	"\x01\x00\x01\x91\x00\x00\x00\x0f\x00\x0d\x00\x00\x0a\x77" +
	"\x77\x77\x2e\x77\x33\x2e\x6f\x72\x67\x00\x17\x00\x00\xff" +
	"\x01\x00\x01\x00\x00\x0a\x00\x0e\x00\x0c\x00\x1d\x00\x17" +
	"\x00\x18\x00\x19\x01\x00\x01\x01\x00\x0b\x00\x02\x01\x00" +
	"\x00\x23\x00\x00\x00\x10\x00\x0e\x00\x0c\x02\x68\x32\x08" +
	"\x68\x74\x74\x70\x2f\x31\x2e\x31\x00\x05\x00\x05\x01\x00" +
	"\x00\x00\x00\x00\x33\x00\x6b\x00\x69\x00\x1d\x00\x20\xb0" +
	"\xe4\xda\x34\xb4\x29\x8d\xd3\x5c\x70\xd3\xbe\xe8\xa7\x2a" +
	"\x6b\xe4\x11\x19\x8b\x18\x9d\x83\x9a\x49\x7c\x83\x7f\xa9" +
	"\x03\x8c\x3c\x00\x17\x00\x41\x04\x4c\x04\xa4\x71\x4c\x49" +
	"\x75\x55\xd1\x18\x1e\x22\x62\x19\x53\x00\xde\x74\x2f\xb3" +
	"\xde\x13\x54\xe6\x78\x07\x94\x55\x0e\xb2\x6c\xb0\x03\xee" +
	"\x79\xa9\x96\x1e\x0e\x98\x17\x78\x24\x44\x0c\x88\x80\x06" +
	"\x8b\xd4\x80\xbf\x67\x7c\x37\x6a\x5b\x46\x4c\xa7\x98\x6f" +
	"\xb9\x22\x00\x2b\x00\x09\x08\x03\x04\x03\x03\x03\x02\x03" +
	"\x01\x00\x0d\x00\x18\x00\x16\x04\x03\x05\x03\x06\x03\x08" +
	"\x04\x08\x05\x08\x06\x04\x01\x05\x01\x06\x01\x02\x03\x02" +
	"\x01\x00\x2d\x00\x02\x01\x01\x00\x1c\x00\x02\x40\x01\x00" +
	"\x15\x00\x96\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" +
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" +
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" +
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" +
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" +
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" +
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" +
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" +
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" +
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" +
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"

type HTTPSSplitModeType int

var availableHTTPSModes = []string{"sni", "random", "chunk", "first-byte", "none"}

const (
	HTTPSSplitModeSNI HTTPSSplitModeType = iota
	HTTPSSplitModeRandom
	HTTPSSplitModeChunk
	HTTPSSplitModeFirstByte
	HTTPSSplitModeNone
)

func (k HTTPSSplitModeType) String() string {
	return availableHTTPSModes[k]
}

type HTTPSOptions struct {
	Disorder   *bool               `toml:"disorder"    json:"ds,omitempty"`
	FakeCount  *uint8              `toml:"fake-count"  json:"fc,omitempty"`
	FakePacket []byte              `toml:"fake-packet" json:"fp,omitempty"`
	SplitMode  *HTTPSSplitModeType `toml:"split-mode"  json:"sm,omitempty"`
	ChunkSize  *uint8              `toml:"chunk-size"  json:"cs,omitempty"`
	Skip       *bool               `toml:"skip"        json:"sk,omitempty"`
}

func (o *HTTPSOptions) UnmarshalTOML(data any) (err error) {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("'https' must be table type")
	}

	o.Disorder = findFrom(m, "disorder", parseBoolFn(), &err)
	o.FakeCount = findFrom(m, "fake-count", parseIntFn[uint8](checkUint8), &err)
	o.FakePacket = findSliceFrom(m, "fake-packet", parseByteFn(nil), &err)

	splitModeParser := parseStringFn(checkHTTPSSplitMode)
	if p := findFrom(m, "split-mode", splitModeParser, &err); isOk(p, err) {
		o.SplitMode = ptr.FromValue(mustParseHTTPSSplitModeType(*p))
	}

	o.ChunkSize = findFrom(m, "chunk-size", parseIntFn[uint8](checkUint8NonZero), &err)
	o.Skip = findFrom(m, "skip", parseBoolFn(), &err)

	return nil
}

func (o *HTTPSOptions) Clone() *HTTPSOptions {
	if o == nil {
		return nil
	}

	return &HTTPSOptions{
		Disorder:   ptr.Clone(o.Disorder),
		FakeCount:  ptr.Clone(o.FakeCount),
		FakePacket: slices.Clone(o.FakePacket),
		SplitMode:  ptr.Clone(o.SplitMode),
		ChunkSize:  ptr.Clone(o.ChunkSize),
		Skip:       ptr.Clone(o.Skip),
	}
}

func (origin *HTTPSOptions) Merge(overrides *HTTPSOptions) *HTTPSOptions {
	if overrides == nil {
		return origin
	}

	if origin == nil {
		return overrides
	}

	return &HTTPSOptions{
		Disorder:   ptr.CloneOr(overrides.Disorder, origin.Disorder),
		FakeCount:  ptr.CloneOr(overrides.FakeCount, origin.FakeCount),
		FakePacket: ptr.CloneSliceOr(overrides.FakePacket, origin.FakePacket),
		SplitMode:  ptr.CloneOr(overrides.SplitMode, origin.SplitMode),
		ChunkSize:  ptr.CloneOr(overrides.ChunkSize, origin.ChunkSize),
		Skip:       ptr.CloneOr(overrides.Skip, origin.Skip),
	}
}

// ┌────────────────┐
// │ POLICY OPTIONS │
// └────────────────┘
var (
	_ merger[*PolicyOptions] = (*PolicyOptions)(nil)
	_ cloner[*MatchAttrs]    = (*MatchAttrs)(nil)
	_ cloner[*Rule]          = (*Rule)(nil)
)

type PolicyOptions struct {
	Auto      *bool  `toml:"auto"`
	Template  *Rule  `toml:"template"`
	Overrides []Rule `toml:"overries"`
}

func (o *PolicyOptions) UnmarshalTOML(data any) (err error) {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("non-table type policy config")
	}

	o.Auto = findFrom(m, "auto", parseBoolFn(), &err)
	o.Template = findStructFrom[Rule](m, "template", &err)
	o.Overrides = findStructSliceFrom[Rule](m, "overrides", &err)

	return err
}

func (o *PolicyOptions) Clone() *PolicyOptions {
	if o == nil {
		return nil
	}

	overrides := make([]Rule, 0, len(o.Overrides))
	for i := range o.Overrides {
		overrides = append(overrides, *o.Overrides[i].Clone())
	}

	return &PolicyOptions{
		Auto:      ptr.Clone(o.Auto),
		Template:  o.Template.Clone(),
		Overrides: overrides,
	}
}

func (origin *PolicyOptions) Merge(overrides *PolicyOptions) *PolicyOptions {
	if overrides == nil {
		return origin.Clone()
	}

	if origin == nil {
		return overrides.Clone()
	}

	overridesCopy := overrides.Clone()

	merged := origin.Clone()
	merged.Auto = ptr.CloneOr(overrides.Auto, origin.Auto)

	if overridesCopy.Template != nil {
		merged.Template = overridesCopy.Template
	}

	if overridesCopy.Overrides != nil {
		merged.Overrides = append(merged.Overrides, overridesCopy.Overrides...)
	}

	return merged
}

type MatchAttrs struct {
	Domain   *string    `toml:"domain" json:"do,omitempty"`
	CIDR     *net.IPNet `toml:"cidr"   json:"cd,omitempty"`
	PortFrom *uint16    `toml:"port"   json:"pf,omitempty"`
	PortTo   *uint16    `toml:"port"   json:"pt,omitempty"`
}

func (a *MatchAttrs) UnmarshalTOML(data any) (err error) {
	v, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("'match' must be table type")
	}

	a.Domain = findFrom(v, "domain", parseStringFn(checkDomainPattern), &err)

	if p := findFrom(v, "cidr", parseStringFn(checkCIDR), &err); isOk(p, err) {
		a.CIDR = ptr.FromValue(MustParseCIDR(*p))
	}

	if p := findFrom(v, "port", parseStringFn(checkPortRange), &err); isOk(p, err) {
		portFrom, portTo := MustParsePortRange(*p)
		a.PortFrom, a.PortTo = ptr.FromValue(portFrom), ptr.FromValue(portTo)
	}

	if err == nil {
		err = checkMatchAttrs(*a)
	}

	return err
}

func (a *MatchAttrs) Clone() *MatchAttrs {
	if a == nil {
		return nil
	}

	return &MatchAttrs{
		Domain:   ptr.Clone(a.Domain),
		CIDR:     ptr.Clone(a.CIDR),
		PortFrom: ptr.Clone(a.PortFrom),
		PortTo:   ptr.Clone(a.PortTo),
	}
}

type Rule struct {
	Name     *string       `toml:"name"           json:"nm,omitempty"`
	Priority *uint16       `toml:"priority"       json:"pr,omitempty"`
	Block    *bool         `toml:"block"          json:"bk,omitempty"`
	Match    *MatchAttrs   `toml:"match"          json:"mt,omitempty"`
	DNS      *DNSOptions   `toml:"dns-override"   json:"D,omitempty"`
	HTTPS    *HTTPSOptions `toml:"https-override" json:"H,omitempty"`
}

func (r *Rule) UnmarshalTOML(data any) (err error) {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("'rule' must be table type")
	}

	r.Name = findFrom(m, "name", parseStringFn(nil), &err)
	r.Priority = findFrom(m, "priority", parseIntFn[uint16](checkUint16), &err)
	r.Block = findFrom(m, "block", parseBoolFn(), &err)
	r.Match = findStructFrom[MatchAttrs](m, "match", &err)
	r.DNS = findStructFrom[DNSOptions](m, "dns", &err)
	r.HTTPS = findStructFrom[HTTPSOptions](m, "https", &err)

	if err == nil {
		err = checkRule(*r)
	}

	return
}

func (r *Rule) Clone() *Rule {
	if r == nil {
		return nil
	}
	return &Rule{
		Name:     ptr.Clone(r.Name),
		Priority: ptr.Clone(r.Priority),
		Block:    ptr.Clone(r.Block),
		Match:    ptr.Clone(r.Match),
		DNS:      r.DNS.Clone(),
		HTTPS:    r.HTTPS.Clone(),
	}
}
