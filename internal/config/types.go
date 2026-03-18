package config

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"github.com/xvzc/SpoofDPI/internal/proto"
)

type primitive interface {
	~bool | ~string |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~complex64 | ~complex128
}

func clonePrimitive[T primitive](x *T) *T {
	if x == nil {
		return nil
	}
	return lo.ToPtr(lo.FromPtr(x))
}

// ┌─────────────────┐
// │ GENERAL OPTIONS │
// └─────────────────┘
var _ merger[*AppOptions] = (*AppOptions)(nil)

var availableLogLevelValues = []string{
	"info",
	"warn",
	"trace",
	"error",
	"debug",
	"disabled",
}

type AppOptions struct {
	LogLevel             *zerolog.Level `toml:"log-level"`
	Silent               *bool          `toml:"silent"`
	AutoConfigureNetwork *bool          `toml:"auto-configure-network"`
	Mode                 *AppModeType   `toml:"mode"`
	ListenAddr           *net.TCPAddr   `toml:"listen-addr"`
}

func (o *AppOptions) UnmarshalTOML(data any) (err error) {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("non-table type general config")
	}

	o.Silent = findFrom(m, "silent", parseBoolFn(), &err)
	o.AutoConfigureNetwork = findFrom(m, "auto-configure-network", parseBoolFn(), &err)
	if p := findFrom(m, "log-level", parseStringFn(checkLogLevel), &err); isOk(p, err) {
		o.LogLevel = lo.ToPtr(MustParseLogLevel(*p))
	}
	if p := findFrom(m, "mode", parseStringFn(checkAppMode), &err); isOk(p, err) {
		o.Mode = lo.ToPtr(MustParseServerModeType(*p))
	}
	if p := findFrom(m, "listen-addr", parseStringFn(checkHostPort), &err); isOk(p, err) {
		o.ListenAddr = lo.ToPtr(MustParseTCPAddr(*p))
	}

	return err
}

func (o *AppOptions) Clone() *AppOptions {
	if o == nil {
		return nil
	}

	var newLevel *zerolog.Level
	if o.LogLevel != nil {
		newLevel = lo.ToPtr(MustParseLogLevel(strings.ToLower(o.LogLevel.String())))
	}

	var newAddr *net.TCPAddr
	if o.ListenAddr != nil {
		newAddr = &net.TCPAddr{
			IP:   append(net.IP(nil), o.ListenAddr.IP...),
			Port: o.ListenAddr.Port,
			Zone: o.ListenAddr.Zone,
		}
	}

	return &AppOptions{
		LogLevel:             newLevel,
		Silent:               clonePrimitive(o.Silent),
		AutoConfigureNetwork: clonePrimitive(o.AutoConfigureNetwork),
		Mode:                 clonePrimitive(o.Mode),
		ListenAddr:           newAddr,
	}
}

func (origin *AppOptions) Merge(overrides *AppOptions) *AppOptions {
	if overrides == nil {
		return origin.Clone()
	}

	if origin == nil {
		return overrides.Clone()
	}

	return &AppOptions{
		LogLevel: lo.CoalesceOrEmpty(overrides.LogLevel, origin.LogLevel),
		Silent:   lo.CoalesceOrEmpty(overrides.Silent, origin.Silent),
		AutoConfigureNetwork: lo.CoalesceOrEmpty(
			overrides.AutoConfigureNetwork,
			origin.AutoConfigureNetwork,
		),
		Mode:       lo.CoalesceOrEmpty(overrides.Mode, origin.Mode),
		ListenAddr: lo.CoalesceOrEmpty(overrides.ListenAddr, origin.ListenAddr),
	}
}

// ┌──────────────────────┐
// │ CONNECTION OPTIONS   │
// └──────────────────────┘
var _ merger[*ConnOptions] = (*ConnOptions)(nil)

type AppModeType int

const (
	AppModeHTTP AppModeType = iota
	AppModeSOCKS5
	AppModeTUN
)

var availableAppModeValues = []string{"http", "socks5", "tun"}

func (t AppModeType) String() string {
	return availableAppModeValues[t]
}

type ConnOptions struct {
	DefaultFakeTTL *uint8         `toml:"default-fake-ttl"`
	DNSTimeout     *time.Duration `toml:"dns-timeout"`
	TCPTimeout     *time.Duration `toml:"tcp-timeout"`
	UDPIdleTimeout *time.Duration `toml:"udp-idle-timeout"`
}

func (o *ConnOptions) UnmarshalTOML(data any) (err error) {
	v, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("non-table type connection config")
	}

	o.DefaultFakeTTL = findFrom(
		v,
		"default-fake-ttl",
		parseIntFn[uint8](checkUint8NonZero),
		&err,
	)

	if p := findFrom(
		v,
		"dns-timeout",
		parseIntFn[uint16](checkUint16),
		&err,
	); isOk(
		p,
		err,
	) {
		o.DNSTimeout = lo.ToPtr(time.Duration(*p) * time.Millisecond)
	}
	if p := findFrom(
		v,
		"tcp-timeout",
		parseIntFn[uint16](checkUint16),
		&err,
	); isOk(
		p,
		err,
	) {
		o.TCPTimeout = lo.ToPtr(time.Duration(*p) * time.Millisecond)
	}
	if p := findFrom(
		v,
		"udp-idle-timeout",
		parseIntFn[uint16](checkUint16),
		&err,
	); isOk(
		p,
		err,
	) {
		o.UDPIdleTimeout = lo.ToPtr(time.Duration(*p) * time.Millisecond)
	}

	return err
}

func (o *ConnOptions) Clone() *ConnOptions {
	if o == nil {
		return nil
	}

	return &ConnOptions{
		DefaultFakeTTL: clonePrimitive(o.DefaultFakeTTL),
		DNSTimeout:     clonePrimitive(o.DNSTimeout),
		TCPTimeout:     clonePrimitive(o.TCPTimeout),
		UDPIdleTimeout: clonePrimitive(o.UDPIdleTimeout),
	}
}

func (origin *ConnOptions) Merge(overrides *ConnOptions) *ConnOptions {
	if overrides == nil {
		return origin.Clone()
	}

	if origin == nil {
		return overrides.Clone()
	}

	return &ConnOptions{
		DefaultFakeTTL: lo.CoalesceOrEmpty(overrides.DefaultFakeTTL, origin.DefaultFakeTTL),
		DNSTimeout:     lo.CoalesceOrEmpty(overrides.DNSTimeout, origin.DNSTimeout),
		TCPTimeout:     lo.CoalesceOrEmpty(overrides.TCPTimeout, origin.TCPTimeout),
		UDPIdleTimeout: lo.CoalesceOrEmpty(overrides.UDPIdleTimeout, origin.UDPIdleTimeout),
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
	availableDNSModeValues  = []string{"udp", "https", "system"}
	availableDNSQueryValues = []string{"ipv4", "ipv6", "all"}
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
	return availableDNSModeValues[t]
}

func (t DNSQueryType) String() string {
	return availableDNSQueryValues[t]
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
		o.Mode = lo.ToPtr(MustParseDNSModeType(*p))
	}

	if p := findFrom(m, "addr", parseStringFn(checkHostPort), &err); isOk(p, err) {
		o.Addr = lo.ToPtr(MustParseTCPAddr(*p))
	}

	o.HTTPSURL = findFrom(m, "https-url", parseStringFn(checkHTTPSEndpoint), &err)

	if p := findFrom(m, "qtype", parseStringFn(checkDNSQueryType), &err); isOk(p, err) {
		o.QType = lo.ToPtr(MustParseDNSQueryType(*p))
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
		Mode:     clonePrimitive(o.Mode),
		Addr:     newAddr,
		HTTPSURL: clonePrimitive(o.HTTPSURL),
		QType:    clonePrimitive(o.QType),
		Cache:    clonePrimitive(o.Cache),
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
		Mode:     lo.CoalesceOrEmpty(overrides.Mode, origin.Mode),
		Addr:     lo.CoalesceOrEmpty(overrides.Addr, origin.Addr),
		HTTPSURL: lo.CoalesceOrEmpty(overrides.HTTPSURL, origin.HTTPSURL),
		QType:    lo.CoalesceOrEmpty(overrides.QType, origin.QType),
		Cache:    lo.CoalesceOrEmpty(overrides.Cache, origin.Cache),
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

var availableHTTPSModeValues = []string{
	"sni",
	"random",
	"chunk",
	"first-byte",
	"custom",
	"none",
}

const (
	HTTPSSplitModeSNI HTTPSSplitModeType = iota
	HTTPSSplitModeRandom
	HTTPSSplitModeChunk
	HTTPSSplitModeFirstByte
	HTTPSSplitModeCustom
	HTTPSSplitModeNone
)

func (k HTTPSSplitModeType) String() string {
	return availableHTTPSModeValues[k]
}

type SegmentFromType int

var availableSegmentFromValues = []string{"head", "sni"}

const (
	SegmentFromHead SegmentFromType = iota
	SegmentFromSNI
)

func (s SegmentFromType) String() string {
	return availableSegmentFromValues[s]
}

type SegmentPlan struct {
	From  SegmentFromType `toml:"from"`
	At    int             `toml:"at"`
	Lazy  bool            `toml:"lazy"`
	Noise int             `toml:"noise"`
}

func (s *SegmentPlan) UnmarshalTOML(data any) (err error) {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("segment must be table type")
	}

	if _, ok := m["from"]; !ok {
		return fmt.Errorf("field 'from' is required")
	}
	if p := findFrom(m, "from", parseStringFn(checkSegmentFrom), &err); isOk(p, err) {
		s.From = mustParseSegmentFromType(*p)
	}

	if _, ok := m["at"]; !ok {
		return fmt.Errorf("field 'at' is required")
	}
	if p := findFrom(m, "at", parseIntFn[int](nil), &err); isOk(p, err) {
		s.At = *p
	}

	if p := findFrom(m, "lazy", parseBoolFn(), &err); isOk(p, err) {
		s.Lazy = *p
	}

	if p := findFrom(m, "noise", parseIntFn[int](nil), &err); isOk(p, err) {
		s.Noise = *p
	}

	return err
}

func (s *SegmentPlan) Clone() *SegmentPlan {
	if s == nil {
		return nil
	}

	return &SegmentPlan{
		From:  s.From,
		At:    s.At,
		Lazy:  s.Lazy,
		Noise: s.Noise,
	}
}

type HTTPSOptions struct {
	Disorder           *bool               `toml:"disorder"        json:"ds,omitempty"`
	FakeCount          *uint8              `toml:"fake-count"      json:"fc,omitempty"`
	FakePacket         *proto.TLSMessage   `toml:"fake-packet"     json:"fp,omitempty"`
	SplitMode          *HTTPSSplitModeType `toml:"split-mode"      json:"sm,omitempty"`
	ChunkSize          *uint8              `toml:"chunk-size"      json:"cs,omitempty"`
	Skip               *bool               `toml:"skip"            json:"sk,omitempty"`
	CustomSegmentPlans []SegmentPlan       `toml:"custom-segments" json:"cseg,omitempty"`
}

func (o *HTTPSOptions) UnmarshalTOML(data any) (err error) {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("'https' must be table type")
	}

	o.Disorder = findFrom(m, "disorder", parseBoolFn(), &err)
	o.FakeCount = findFrom(m, "fake-count", parseIntFn[uint8](checkUint8), &err)

	fakePacket := findSliceFrom(m, "fake-packet", parseByteFn(nil), &err)
	if fakePacket != nil {
		o.FakePacket = proto.NewFakeTLSMessage(fakePacket)
	}

	splitModeParser := parseStringFn(checkHTTPSSplitMode)
	if p := findFrom(m, "split-mode", splitModeParser, &err); isOk(p, err) {
		o.SplitMode = lo.ToPtr(mustParseHTTPSSplitModeType(*p))
	}

	o.ChunkSize = findFrom(m, "chunk-size", parseIntFn[uint8](checkUint8NonZero), &err)
	o.Skip = findFrom(m, "skip", parseBoolFn(), &err)
	if o.Skip == nil {
		o.Skip = lo.ToPtr(false)
	}

	o.CustomSegmentPlans = findStructSliceFrom[SegmentPlan](m, "custom-segments", &err)
	if err == nil && o.SplitMode != nil && *o.SplitMode == HTTPSSplitModeCustom &&
		len(o.CustomSegmentPlans) == 0 {
		err = fmt.Errorf("custom-segments must be provided when split-mode is 'custom'")
	}

	return err
}

func (o *HTTPSOptions) Clone() *HTTPSOptions {
	if o == nil {
		return nil
	}

	var fakePacket *proto.TLSMessage
	if o.FakePacket != nil {
		fakePacket = proto.NewFakeTLSMessage(o.FakePacket.Raw())
	}

	var customSegmentPlans []SegmentPlan
	if o.CustomSegmentPlans != nil {
		customSegmentPlans = make([]SegmentPlan, 0, len(o.CustomSegmentPlans))
		for _, s := range o.CustomSegmentPlans {
			customSegmentPlans = append(customSegmentPlans, *s.Clone())
		}
	}

	return &HTTPSOptions{
		Disorder:           clonePrimitive(o.Disorder),
		FakeCount:          clonePrimitive(o.FakeCount),
		FakePacket:         fakePacket,
		SplitMode:          clonePrimitive(o.SplitMode),
		ChunkSize:          clonePrimitive(o.ChunkSize),
		Skip:               clonePrimitive(o.Skip),
		CustomSegmentPlans: customSegmentPlans,
	}
}

func (origin *HTTPSOptions) Merge(overrides *HTTPSOptions) *HTTPSOptions {
	if overrides == nil {
		return origin.Clone()
	}

	if origin == nil {
		return overrides.Clone()
	}

	return &HTTPSOptions{
		Disorder:   lo.CoalesceOrEmpty(overrides.Disorder, origin.Disorder),
		FakeCount:  lo.CoalesceOrEmpty(overrides.FakeCount, origin.FakeCount),
		FakePacket: lo.CoalesceOrEmpty(overrides.FakePacket, origin.FakePacket),
		SplitMode:  lo.CoalesceOrEmpty(overrides.SplitMode, origin.SplitMode),
		ChunkSize:  lo.CoalesceOrEmpty(overrides.ChunkSize, origin.ChunkSize),
		Skip:       lo.CoalesceOrEmpty(overrides.Skip, origin.Skip),
		CustomSegmentPlans: lo.CoalesceSliceOrEmpty(
			origin.CustomSegmentPlans,
			overrides.CustomSegmentPlans,
		),
	}
}

// ┌─────────────┐
// │ UDP OPTIONS │
// └─────────────┘
var _ merger[*UDPOptions] = (*UDPOptions)(nil)

type UDPOptions struct {
	FakeCount  *int   `toml:"fake-count"  json:"fc,omitempty"`
	FakePacket []byte `toml:"fake-packet" json:"fp,omitempty"`
}

func (o *UDPOptions) UnmarshalTOML(data any) (err error) {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("'udp' must be table type")
	}

	o.FakeCount = findFrom(
		m, "fake-count", parseIntFn[int](int64Range(0, math.MaxInt64)), &err,
	)
	o.FakePacket = findSliceFrom(m, "fake-packet", parseByteFn(nil), &err)

	return err
}

func (o *UDPOptions) Clone() *UDPOptions {
	if o == nil {
		return nil
	}

	return &UDPOptions{
		FakeCount:  clonePrimitive(o.FakeCount),
		FakePacket: append([]byte(nil), o.FakePacket...),
	}
}

func (origin *UDPOptions) Merge(overrides *UDPOptions) *UDPOptions {
	if overrides == nil {
		return origin.Clone()
	}

	if origin == nil {
		return overrides.Clone()
	}

	fakePacket := origin.FakePacket
	if len(overrides.FakePacket) > 0 {
		fakePacket = overrides.FakePacket
	}

	return &UDPOptions{
		FakeCount:  lo.CoalesceOrEmpty(overrides.FakeCount, origin.FakeCount),
		FakePacket: fakePacket,
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
	Template  *Rule  `toml:"template"`
	Overrides []Rule `toml:"overries"`
}

func (o *PolicyOptions) UnmarshalTOML(data any) (err error) {
	m, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("non-table type policy config")
	}

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

	return &PolicyOptions{
		Template:  lo.CoalesceOrEmpty(overrides.Template.Clone(), origin.Template.Clone()),
		Overrides: lo.CoalesceSliceOrEmpty(overrides.Overrides, origin.Overrides),
	}
}

type AddrMatch struct {
	CIDR     *net.IPNet `toml:"cidr" json:"cd,omitempty"`
	PortFrom *uint16    `toml:"port" json:"pf,omitempty"`
	PortTo   *uint16    `toml:"port" json:"pt,omitempty"`
}

func (a *AddrMatch) UnmarshalTOML(data any) (err error) {
	v, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("addr rule must be table type")
	}

	if p := findFrom(v, "cidr", parseStringFn(checkCIDR), &err); isOk(p, err) {
		a.CIDR = lo.ToPtr(MustParseCIDR(*p))
	}

	if p := findFrom(v, "port", parseStringFn(checkPortRange), &err); isOk(p, err) {
		portFrom, portTo := MustParsePortRange(*p)
		a.PortFrom, a.PortTo = lo.ToPtr(portFrom), lo.ToPtr(portTo)
	}

	return err
}

func (a *AddrMatch) Clone() *AddrMatch {
	if a == nil {
		return nil
	}

	var cidr *net.IPNet
	if a.CIDR != nil {
		cidr = &net.IPNet{
			IP:   slices.Clone(a.CIDR.IP),
			Mask: slices.Clone(a.CIDR.Mask),
		}
	}

	return &AddrMatch{
		CIDR:     cidr,
		PortFrom: clonePrimitive(a.PortFrom),
		PortTo:   clonePrimitive(a.PortTo),
	}
}

type MatchAttrs struct {
	Domains []string    `toml:"domain" json:"do,omitempty"`
	Addrs   []AddrMatch `toml:"addr"   json:"ad,omitempty"`
}

func (a *MatchAttrs) UnmarshalTOML(data any) (err error) {
	v, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("'match' must be table type")
	}

	a.Domains = findSliceFrom(v, "domain", parseStringFn(checkDomainPattern), &err)
	a.Addrs = findStructSliceFrom[AddrMatch](v, "addr", &err)

	if err == nil {
		err = checkMatchAttrs(*a)
	}

	return err
}

func (a *MatchAttrs) Clone() *MatchAttrs {
	if a == nil {
		return nil
	}

	addrs := make([]AddrMatch, 0, len(a.Addrs))
	for _, addr := range a.Addrs {
		addrs = append(addrs, *addr.Clone())
	}

	return &MatchAttrs{
		Domains: lo.CoalesceSliceOrEmpty(a.Domains),
		Addrs:   addrs,
	}
}

type Rule struct {
	Name     *string       `toml:"name"           json:"nm,omitempty"`
	Priority *uint16       `toml:"priority"       json:"pr,omitempty"`
	Block    *bool         `toml:"block"          json:"bk,omitempty"`
	Match    *MatchAttrs   `toml:"match"          json:"mt,omitempty"`
	DNS      *DNSOptions   `toml:"dns-override"   json:"D,omitempty"`
	HTTPS    *HTTPSOptions `toml:"https-override" json:"H,omitempty"`
	UDP      *UDPOptions   `toml:"udp-override"   json:"U,omitempty"`
	Conn     *ConnOptions  `toml:"conn-override"  json:"C,omitempty"`
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
	r.UDP = findStructFrom[UDPOptions](m, "udp", &err)
	r.Conn = findStructFrom[ConnOptions](m, "connection", &err)

	// if err == nil {
	// 	err = checkRule(*r)
	// }

	return
}

func (r *Rule) Clone() *Rule {
	if r == nil {
		return nil
	}
	return &Rule{
		Name:     clonePrimitive(r.Name),
		Priority: clonePrimitive(r.Priority),
		Block:    clonePrimitive(r.Block),
		Match:    r.Match.Clone(),
		DNS:      r.DNS.Clone(),
		HTTPS:    r.HTTPS.Clone(),
		UDP:      r.UDP.Clone(),
		Conn:     r.Conn.Clone(),
	}
}

func (r *Rule) JSON() []byte {
	data := map[string]any{
		"name":     r.Name,
		"priority": r.Priority,
	}

	if r.Match == nil {
		data["match"] = nil
	} else {
		m := map[string]any{}
		if r.Match.Addrs != nil {
			m["addr"] = fmt.Sprintf("%v items", len(r.Match.Addrs))
		}
		if r.Match.Domains != nil {
			m["domain"] = fmt.Sprintf("%v items", len(r.Match.Domains))
		}
		data["match"] = m
	}

	bytes, _ := json.Marshal(data)
	return bytes
}
