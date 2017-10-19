package porto

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *Profile) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zxhx uint32
	zxhx, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zxhx > 0 {
		zxhx--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "registry":
			z.Registry, err = dc.ReadString()
			if err != nil {
				return
			}
		case "repository":
			z.Repository, err = dc.ReadString()
			if err != nil {
				return
			}
		case "network_mode":
			z.NetworkMode, err = dc.ReadString()
			if err != nil {
				return
			}
		case "network":
			var zlqf uint32
			zlqf, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			if z.Network == nil && zlqf > 0 {
				z.Network = make(map[string]string, zlqf)
			} else if len(z.Network) > 0 {
				for key, _ := range z.Network {
					delete(z.Network, key)
				}
			}
			for zlqf > 0 {
				zlqf--
				var zxvk string
				var zbzg string
				zxvk, err = dc.ReadString()
				if err != nil {
					return
				}
				zbzg, err = dc.ReadString()
				if err != nil {
					return
				}
				z.Network[zxvk] = zbzg
			}
		case "cwd":
			z.Cwd, err = dc.ReadString()
			if err != nil {
				return
			}
		case "binds":
			var zdaf uint32
			zdaf, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Binds) >= int(zdaf) {
				z.Binds = (z.Binds)[:zdaf]
			} else {
				z.Binds = make([]string, zdaf)
			}
			for zbai := range z.Binds {
				z.Binds[zbai], err = dc.ReadString()
				if err != nil {
					return
				}
			}
		case "container":
			var zpks uint32
			zpks, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			if z.Container == nil && zpks > 0 {
				z.Container = make(map[string]string, zpks)
			} else if len(z.Container) > 0 {
				for key, _ := range z.Container {
					delete(z.Container, key)
				}
			}
			for zpks > 0 {
				zpks--
				var zcmr string
				var zajw string
				zcmr, err = dc.ReadString()
				if err != nil {
					return
				}
				zajw, err = dc.ReadString()
				if err != nil {
					return
				}
				z.Container[zcmr] = zajw
			}
		case "volume":
			var zjfb uint32
			zjfb, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			if z.Volume == nil && zjfb > 0 {
				z.Volume = make(map[string]string, zjfb)
			} else if len(z.Volume) > 0 {
				for key, _ := range z.Volume {
					delete(z.Volume, key)
				}
			}
			for zjfb > 0 {
				zjfb--
				var zwht string
				var zhct string
				zwht, err = dc.ReadString()
				if err != nil {
					return
				}
				zhct, err = dc.ReadString()
				if err != nil {
					return
				}
				z.Volume[zwht] = zhct
			}
		case "extravolumes":
			var zcxo uint32
			zcxo, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.ExtraVolumes) >= int(zcxo) {
				z.ExtraVolumes = (z.ExtraVolumes)[:zcxo]
			} else {
				z.ExtraVolumes = make([]VolumeProfile, zcxo)
			}
			for zcua := range z.ExtraVolumes {
				err = z.ExtraVolumes[zcua].DecodeMsg(dc)
				if err != nil {
					return
				}
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Profile) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 9
	// write "registry"
	err = en.Append(0x89, 0xa8, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Registry)
	if err != nil {
		return
	}
	// write "repository"
	err = en.Append(0xaa, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Repository)
	if err != nil {
		return
	}
	// write "network_mode"
	err = en.Append(0xac, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b, 0x5f, 0x6d, 0x6f, 0x64, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteString(z.NetworkMode)
	if err != nil {
		return
	}
	// write "network"
	err = en.Append(0xa7, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b)
	if err != nil {
		return err
	}
	err = en.WriteMapHeader(uint32(len(z.Network)))
	if err != nil {
		return
	}
	for zxvk, zbzg := range z.Network {
		err = en.WriteString(zxvk)
		if err != nil {
			return
		}
		err = en.WriteString(zbzg)
		if err != nil {
			return
		}
	}
	// write "cwd"
	err = en.Append(0xa3, 0x63, 0x77, 0x64)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Cwd)
	if err != nil {
		return
	}
	// write "binds"
	err = en.Append(0xa5, 0x62, 0x69, 0x6e, 0x64, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.Binds)))
	if err != nil {
		return
	}
	for zbai := range z.Binds {
		err = en.WriteString(z.Binds[zbai])
		if err != nil {
			return
		}
	}
	// write "container"
	err = en.Append(0xa9, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72)
	if err != nil {
		return err
	}
	err = en.WriteMapHeader(uint32(len(z.Container)))
	if err != nil {
		return
	}
	for zcmr, zajw := range z.Container {
		err = en.WriteString(zcmr)
		if err != nil {
			return
		}
		err = en.WriteString(zajw)
		if err != nil {
			return
		}
	}
	// write "volume"
	err = en.Append(0xa6, 0x76, 0x6f, 0x6c, 0x75, 0x6d, 0x65)
	if err != nil {
		return err
	}
	err = en.WriteMapHeader(uint32(len(z.Volume)))
	if err != nil {
		return
	}
	for zwht, zhct := range z.Volume {
		err = en.WriteString(zwht)
		if err != nil {
			return
		}
		err = en.WriteString(zhct)
		if err != nil {
			return
		}
	}
	// write "extravolumes"
	err = en.Append(0xac, 0x65, 0x78, 0x74, 0x72, 0x61, 0x76, 0x6f, 0x6c, 0x75, 0x6d, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteArrayHeader(uint32(len(z.ExtraVolumes)))
	if err != nil {
		return
	}
	for zcua := range z.ExtraVolumes {
		err = z.ExtraVolumes[zcua].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Profile) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 9
	// string "registry"
	o = append(o, 0x89, 0xa8, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79)
	o = msgp.AppendString(o, z.Registry)
	// string "repository"
	o = append(o, 0xaa, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79)
	o = msgp.AppendString(o, z.Repository)
	// string "network_mode"
	o = append(o, 0xac, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b, 0x5f, 0x6d, 0x6f, 0x64, 0x65)
	o = msgp.AppendString(o, z.NetworkMode)
	// string "network"
	o = append(o, 0xa7, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b)
	o = msgp.AppendMapHeader(o, uint32(len(z.Network)))
	for zxvk, zbzg := range z.Network {
		o = msgp.AppendString(o, zxvk)
		o = msgp.AppendString(o, zbzg)
	}
	// string "cwd"
	o = append(o, 0xa3, 0x63, 0x77, 0x64)
	o = msgp.AppendString(o, z.Cwd)
	// string "binds"
	o = append(o, 0xa5, 0x62, 0x69, 0x6e, 0x64, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Binds)))
	for zbai := range z.Binds {
		o = msgp.AppendString(o, z.Binds[zbai])
	}
	// string "container"
	o = append(o, 0xa9, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72)
	o = msgp.AppendMapHeader(o, uint32(len(z.Container)))
	for zcmr, zajw := range z.Container {
		o = msgp.AppendString(o, zcmr)
		o = msgp.AppendString(o, zajw)
	}
	// string "volume"
	o = append(o, 0xa6, 0x76, 0x6f, 0x6c, 0x75, 0x6d, 0x65)
	o = msgp.AppendMapHeader(o, uint32(len(z.Volume)))
	for zwht, zhct := range z.Volume {
		o = msgp.AppendString(o, zwht)
		o = msgp.AppendString(o, zhct)
	}
	// string "extravolumes"
	o = append(o, 0xac, 0x65, 0x78, 0x74, 0x72, 0x61, 0x76, 0x6f, 0x6c, 0x75, 0x6d, 0x65, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.ExtraVolumes)))
	for zcua := range z.ExtraVolumes {
		o, err = z.ExtraVolumes[zcua].MarshalMsg(o)
		if err != nil {
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Profile) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zeff uint32
	zeff, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zeff > 0 {
		zeff--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "registry":
			z.Registry, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "repository":
			z.Repository, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "network_mode":
			z.NetworkMode, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "network":
			var zrsw uint32
			zrsw, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				return
			}
			if z.Network == nil && zrsw > 0 {
				z.Network = make(map[string]string, zrsw)
			} else if len(z.Network) > 0 {
				for key, _ := range z.Network {
					delete(z.Network, key)
				}
			}
			for zrsw > 0 {
				var zxvk string
				var zbzg string
				zrsw--
				zxvk, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				zbzg, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				z.Network[zxvk] = zbzg
			}
		case "cwd":
			z.Cwd, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "binds":
			var zxpk uint32
			zxpk, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Binds) >= int(zxpk) {
				z.Binds = (z.Binds)[:zxpk]
			} else {
				z.Binds = make([]string, zxpk)
			}
			for zbai := range z.Binds {
				z.Binds[zbai], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
			}
		case "container":
			var zdnj uint32
			zdnj, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				return
			}
			if z.Container == nil && zdnj > 0 {
				z.Container = make(map[string]string, zdnj)
			} else if len(z.Container) > 0 {
				for key, _ := range z.Container {
					delete(z.Container, key)
				}
			}
			for zdnj > 0 {
				var zcmr string
				var zajw string
				zdnj--
				zcmr, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				zajw, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				z.Container[zcmr] = zajw
			}
		case "volume":
			var zobc uint32
			zobc, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				return
			}
			if z.Volume == nil && zobc > 0 {
				z.Volume = make(map[string]string, zobc)
			} else if len(z.Volume) > 0 {
				for key, _ := range z.Volume {
					delete(z.Volume, key)
				}
			}
			for zobc > 0 {
				var zwht string
				var zhct string
				zobc--
				zwht, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				zhct, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				z.Volume[zwht] = zhct
			}
		case "extravolumes":
			var zsnv uint32
			zsnv, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.ExtraVolumes) >= int(zsnv) {
				z.ExtraVolumes = (z.ExtraVolumes)[:zsnv]
			} else {
				z.ExtraVolumes = make([]VolumeProfile, zsnv)
			}
			for zcua := range z.ExtraVolumes {
				bts, err = z.ExtraVolumes[zcua].UnmarshalMsg(bts)
				if err != nil {
					return
				}
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Profile) Msgsize() (s int) {
	s = 1 + 9 + msgp.StringPrefixSize + len(z.Registry) + 11 + msgp.StringPrefixSize + len(z.Repository) + 13 + msgp.StringPrefixSize + len(z.NetworkMode) + 8 + msgp.MapHeaderSize
	if z.Network != nil {
		for zxvk, zbzg := range z.Network {
			_ = zbzg
			s += msgp.StringPrefixSize + len(zxvk) + msgp.StringPrefixSize + len(zbzg)
		}
	}
	s += 4 + msgp.StringPrefixSize + len(z.Cwd) + 6 + msgp.ArrayHeaderSize
	for zbai := range z.Binds {
		s += msgp.StringPrefixSize + len(z.Binds[zbai])
	}
	s += 10 + msgp.MapHeaderSize
	if z.Container != nil {
		for zcmr, zajw := range z.Container {
			_ = zajw
			s += msgp.StringPrefixSize + len(zcmr) + msgp.StringPrefixSize + len(zajw)
		}
	}
	s += 7 + msgp.MapHeaderSize
	if z.Volume != nil {
		for zwht, zhct := range z.Volume {
			_ = zhct
			s += msgp.StringPrefixSize + len(zwht) + msgp.StringPrefixSize + len(zhct)
		}
	}
	s += 13 + msgp.ArrayHeaderSize
	for zcua := range z.ExtraVolumes {
		s += z.ExtraVolumes[zcua].Msgsize()
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *VolumeProfile) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zpez uint32
	zpez, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zpez > 0 {
		zpez--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "target":
			z.Target, err = dc.ReadString()
			if err != nil {
				return
			}
		case "properties":
			var zqke uint32
			zqke, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			if z.Properties == nil && zqke > 0 {
				z.Properties = make(map[string]string, zqke)
			} else if len(z.Properties) > 0 {
				for key, _ := range z.Properties {
					delete(z.Properties, key)
				}
			}
			for zqke > 0 {
				zqke--
				var zkgt string
				var zema string
				zkgt, err = dc.ReadString()
				if err != nil {
					return
				}
				zema, err = dc.ReadString()
				if err != nil {
					return
				}
				z.Properties[zkgt] = zema
			}
		default:
			err = dc.Skip()
			if err != nil {
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *VolumeProfile) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "target"
	err = en.Append(0x82, 0xa6, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Target)
	if err != nil {
		return
	}
	// write "properties"
	err = en.Append(0xaa, 0x70, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteMapHeader(uint32(len(z.Properties)))
	if err != nil {
		return
	}
	for zkgt, zema := range z.Properties {
		err = en.WriteString(zkgt)
		if err != nil {
			return
		}
		err = en.WriteString(zema)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *VolumeProfile) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "target"
	o = append(o, 0x82, 0xa6, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74)
	o = msgp.AppendString(o, z.Target)
	// string "properties"
	o = append(o, 0xaa, 0x70, 0x72, 0x6f, 0x70, 0x65, 0x72, 0x74, 0x69, 0x65, 0x73)
	o = msgp.AppendMapHeader(o, uint32(len(z.Properties)))
	for zkgt, zema := range z.Properties {
		o = msgp.AppendString(o, zkgt)
		o = msgp.AppendString(o, zema)
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *VolumeProfile) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zqyh uint32
	zqyh, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zqyh > 0 {
		zqyh--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "target":
			z.Target, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "properties":
			var zyzr uint32
			zyzr, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				return
			}
			if z.Properties == nil && zyzr > 0 {
				z.Properties = make(map[string]string, zyzr)
			} else if len(z.Properties) > 0 {
				for key, _ := range z.Properties {
					delete(z.Properties, key)
				}
			}
			for zyzr > 0 {
				var zkgt string
				var zema string
				zyzr--
				zkgt, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				zema, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				z.Properties[zkgt] = zema
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *VolumeProfile) Msgsize() (s int) {
	s = 1 + 7 + msgp.StringPrefixSize + len(z.Target) + 11 + msgp.MapHeaderSize
	if z.Properties != nil {
		for zkgt, zema := range z.Properties {
			_ = zema
			s += msgp.StringPrefixSize + len(zkgt) + msgp.StringPrefixSize + len(zema)
		}
	}
	return
}
