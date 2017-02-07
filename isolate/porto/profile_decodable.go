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
	var zhct uint32
	zhct, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zhct > 0 {
		zhct--
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
		case "cwd":
			z.Cwd, err = dc.ReadString()
			if err != nil {
				return
			}
		case "binds":
			var zcua uint32
			zcua, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Binds) >= int(zcua) {
				z.Binds = (z.Binds)[:zcua]
			} else {
				z.Binds = make([]string, zcua)
			}
			for zxvk := range z.Binds {
				z.Binds[zxvk], err = dc.ReadString()
				if err != nil {
					return
				}
			}
		case "container":
			var zxhx uint32
			zxhx, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			if z.Container == nil && zxhx > 0 {
				z.Container = make(map[string]string, zxhx)
			} else if len(z.Container) > 0 {
				for key, _ := range z.Container {
					delete(z.Container, key)
				}
			}
			for zxhx > 0 {
				zxhx--
				var zbzg string
				var zbai string
				zbzg, err = dc.ReadString()
				if err != nil {
					return
				}
				zbai, err = dc.ReadString()
				if err != nil {
					return
				}
				z.Container[zbzg] = zbai
			}
		case "volume":
			var zlqf uint32
			zlqf, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			if z.Volume == nil && zlqf > 0 {
				z.Volume = make(map[string]string, zlqf)
			} else if len(z.Volume) > 0 {
				for key, _ := range z.Volume {
					delete(z.Volume, key)
				}
			}
			for zlqf > 0 {
				zlqf--
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
				z.Volume[zcmr] = zajw
			}
		case "extravolumes":
			var zdaf uint32
			zdaf, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.ExtraVolumes) >= int(zdaf) {
				z.ExtraVolumes = (z.ExtraVolumes)[:zdaf]
			} else {
				z.ExtraVolumes = make([]VolumeProfile, zdaf)
			}
			for zwht := range z.ExtraVolumes {
				err = z.ExtraVolumes[zwht].DecodeMsg(dc)
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
	// map header, size 8
	// write "registry"
	err = en.Append(0x88, 0xa8, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79)
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
	for zxvk := range z.Binds {
		err = en.WriteString(z.Binds[zxvk])
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
	for zbzg, zbai := range z.Container {
		err = en.WriteString(zbzg)
		if err != nil {
			return
		}
		err = en.WriteString(zbai)
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
	for zcmr, zajw := range z.Volume {
		err = en.WriteString(zcmr)
		if err != nil {
			return
		}
		err = en.WriteString(zajw)
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
	for zwht := range z.ExtraVolumes {
		err = z.ExtraVolumes[zwht].EncodeMsg(en)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Profile) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 8
	// string "registry"
	o = append(o, 0x88, 0xa8, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79)
	o = msgp.AppendString(o, z.Registry)
	// string "repository"
	o = append(o, 0xaa, 0x72, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79)
	o = msgp.AppendString(o, z.Repository)
	// string "network_mode"
	o = append(o, 0xac, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b, 0x5f, 0x6d, 0x6f, 0x64, 0x65)
	o = msgp.AppendString(o, z.NetworkMode)
	// string "cwd"
	o = append(o, 0xa3, 0x63, 0x77, 0x64)
	o = msgp.AppendString(o, z.Cwd)
	// string "binds"
	o = append(o, 0xa5, 0x62, 0x69, 0x6e, 0x64, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Binds)))
	for zxvk := range z.Binds {
		o = msgp.AppendString(o, z.Binds[zxvk])
	}
	// string "container"
	o = append(o, 0xa9, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72)
	o = msgp.AppendMapHeader(o, uint32(len(z.Container)))
	for zbzg, zbai := range z.Container {
		o = msgp.AppendString(o, zbzg)
		o = msgp.AppendString(o, zbai)
	}
	// string "volume"
	o = append(o, 0xa6, 0x76, 0x6f, 0x6c, 0x75, 0x6d, 0x65)
	o = msgp.AppendMapHeader(o, uint32(len(z.Volume)))
	for zcmr, zajw := range z.Volume {
		o = msgp.AppendString(o, zcmr)
		o = msgp.AppendString(o, zajw)
	}
	// string "extravolumes"
	o = append(o, 0xac, 0x65, 0x78, 0x74, 0x72, 0x61, 0x76, 0x6f, 0x6c, 0x75, 0x6d, 0x65, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.ExtraVolumes)))
	for zwht := range z.ExtraVolumes {
		o, err = z.ExtraVolumes[zwht].MarshalMsg(o)
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
	var zpks uint32
	zpks, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zpks > 0 {
		zpks--
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
		case "cwd":
			z.Cwd, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "binds":
			var zjfb uint32
			zjfb, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Binds) >= int(zjfb) {
				z.Binds = (z.Binds)[:zjfb]
			} else {
				z.Binds = make([]string, zjfb)
			}
			for zxvk := range z.Binds {
				z.Binds[zxvk], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
			}
		case "container":
			var zcxo uint32
			zcxo, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				return
			}
			if z.Container == nil && zcxo > 0 {
				z.Container = make(map[string]string, zcxo)
			} else if len(z.Container) > 0 {
				for key, _ := range z.Container {
					delete(z.Container, key)
				}
			}
			for zcxo > 0 {
				var zbzg string
				var zbai string
				zcxo--
				zbzg, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				zbai, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				z.Container[zbzg] = zbai
			}
		case "volume":
			var zeff uint32
			zeff, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				return
			}
			if z.Volume == nil && zeff > 0 {
				z.Volume = make(map[string]string, zeff)
			} else if len(z.Volume) > 0 {
				for key, _ := range z.Volume {
					delete(z.Volume, key)
				}
			}
			for zeff > 0 {
				var zcmr string
				var zajw string
				zeff--
				zcmr, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				zajw, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				z.Volume[zcmr] = zajw
			}
		case "extravolumes":
			var zrsw uint32
			zrsw, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.ExtraVolumes) >= int(zrsw) {
				z.ExtraVolumes = (z.ExtraVolumes)[:zrsw]
			} else {
				z.ExtraVolumes = make([]VolumeProfile, zrsw)
			}
			for zwht := range z.ExtraVolumes {
				bts, err = z.ExtraVolumes[zwht].UnmarshalMsg(bts)
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
	s = 1 + 9 + msgp.StringPrefixSize + len(z.Registry) + 11 + msgp.StringPrefixSize + len(z.Repository) + 13 + msgp.StringPrefixSize + len(z.NetworkMode) + 4 + msgp.StringPrefixSize + len(z.Cwd) + 6 + msgp.ArrayHeaderSize
	for zxvk := range z.Binds {
		s += msgp.StringPrefixSize + len(z.Binds[zxvk])
	}
	s += 10 + msgp.MapHeaderSize
	if z.Container != nil {
		for zbzg, zbai := range z.Container {
			_ = zbai
			s += msgp.StringPrefixSize + len(zbzg) + msgp.StringPrefixSize + len(zbai)
		}
	}
	s += 7 + msgp.MapHeaderSize
	if z.Volume != nil {
		for zcmr, zajw := range z.Volume {
			_ = zajw
			s += msgp.StringPrefixSize + len(zcmr) + msgp.StringPrefixSize + len(zajw)
		}
	}
	s += 13 + msgp.ArrayHeaderSize
	for zwht := range z.ExtraVolumes {
		s += z.ExtraVolumes[zwht].Msgsize()
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *VolumeProfile) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zobc uint32
	zobc, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zobc > 0 {
		zobc--
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
			var zsnv uint32
			zsnv, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			if z.Properties == nil && zsnv > 0 {
				z.Properties = make(map[string]string, zsnv)
			} else if len(z.Properties) > 0 {
				for key, _ := range z.Properties {
					delete(z.Properties, key)
				}
			}
			for zsnv > 0 {
				zsnv--
				var zxpk string
				var zdnj string
				zxpk, err = dc.ReadString()
				if err != nil {
					return
				}
				zdnj, err = dc.ReadString()
				if err != nil {
					return
				}
				z.Properties[zxpk] = zdnj
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
	for zxpk, zdnj := range z.Properties {
		err = en.WriteString(zxpk)
		if err != nil {
			return
		}
		err = en.WriteString(zdnj)
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
	for zxpk, zdnj := range z.Properties {
		o = msgp.AppendString(o, zxpk)
		o = msgp.AppendString(o, zdnj)
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *VolumeProfile) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zkgt uint32
	zkgt, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zkgt > 0 {
		zkgt--
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
			var zema uint32
			zema, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				return
			}
			if z.Properties == nil && zema > 0 {
				z.Properties = make(map[string]string, zema)
			} else if len(z.Properties) > 0 {
				for key, _ := range z.Properties {
					delete(z.Properties, key)
				}
			}
			for zema > 0 {
				var zxpk string
				var zdnj string
				zema--
				zxpk, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				zdnj, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				z.Properties[zxpk] = zdnj
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
		for zxpk, zdnj := range z.Properties {
			_ = zdnj
			s += msgp.StringPrefixSize + len(zxpk) + msgp.StringPrefixSize + len(zdnj)
		}
	}
	return
}
