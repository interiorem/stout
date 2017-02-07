package docker

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
	var zcmr uint32
	zcmr, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zcmr > 0 {
		zcmr--
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
		case "endpoint":
			z.Endpoint, err = dc.ReadString()
			if err != nil {
				return
			}
		case "network_mode":
			z.NetworkMode, err = dc.ReadString()
			if err != nil {
				return
			}
		case "runtime-path":
			z.RuntimePath, err = dc.ReadString()
			if err != nil {
				return
			}
		case "cwd":
			z.Cwd, err = dc.ReadString()
			if err != nil {
				return
			}
		case "resources":
			err = z.Resources.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "tmpfs":
			var zajw uint32
			zajw, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			if z.Tmpfs == nil && zajw > 0 {
				z.Tmpfs = make(map[string]string, zajw)
			} else if len(z.Tmpfs) > 0 {
				for key, _ := range z.Tmpfs {
					delete(z.Tmpfs, key)
				}
			}
			for zajw > 0 {
				zajw--
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
				z.Tmpfs[zxvk] = zbzg
			}
		case "binds":
			var zwht uint32
			zwht, err = dc.ReadArrayHeader()
			if err != nil {
				return
			}
			if cap(z.Binds) >= int(zwht) {
				z.Binds = (z.Binds)[:zwht]
			} else {
				z.Binds = make([]string, zwht)
			}
			for zbai := range z.Binds {
				z.Binds[zbai], err = dc.ReadString()
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
	// write "endpoint"
	err = en.Append(0xa8, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74)
	if err != nil {
		return err
	}
	err = en.WriteString(z.Endpoint)
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
	// write "runtime-path"
	err = en.Append(0xac, 0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x2d, 0x70, 0x61, 0x74, 0x68)
	if err != nil {
		return err
	}
	err = en.WriteString(z.RuntimePath)
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
	// write "resources"
	err = en.Append(0xa9, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = z.Resources.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "tmpfs"
	err = en.Append(0xa5, 0x74, 0x6d, 0x70, 0x66, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteMapHeader(uint32(len(z.Tmpfs)))
	if err != nil {
		return
	}
	for zxvk, zbzg := range z.Tmpfs {
		err = en.WriteString(zxvk)
		if err != nil {
			return
		}
		err = en.WriteString(zbzg)
		if err != nil {
			return
		}
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
	// string "endpoint"
	o = append(o, 0xa8, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69, 0x6e, 0x74)
	o = msgp.AppendString(o, z.Endpoint)
	// string "network_mode"
	o = append(o, 0xac, 0x6e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b, 0x5f, 0x6d, 0x6f, 0x64, 0x65)
	o = msgp.AppendString(o, z.NetworkMode)
	// string "runtime-path"
	o = append(o, 0xac, 0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x2d, 0x70, 0x61, 0x74, 0x68)
	o = msgp.AppendString(o, z.RuntimePath)
	// string "cwd"
	o = append(o, 0xa3, 0x63, 0x77, 0x64)
	o = msgp.AppendString(o, z.Cwd)
	// string "resources"
	o = append(o, 0xa9, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x73)
	o, err = z.Resources.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "tmpfs"
	o = append(o, 0xa5, 0x74, 0x6d, 0x70, 0x66, 0x73)
	o = msgp.AppendMapHeader(o, uint32(len(z.Tmpfs)))
	for zxvk, zbzg := range z.Tmpfs {
		o = msgp.AppendString(o, zxvk)
		o = msgp.AppendString(o, zbzg)
	}
	// string "binds"
	o = append(o, 0xa5, 0x62, 0x69, 0x6e, 0x64, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Binds)))
	for zbai := range z.Binds {
		o = msgp.AppendString(o, z.Binds[zbai])
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Profile) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zhct uint32
	zhct, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zhct > 0 {
		zhct--
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
		case "endpoint":
			z.Endpoint, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "network_mode":
			z.NetworkMode, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "runtime-path":
			z.RuntimePath, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "cwd":
			z.Cwd, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "resources":
			bts, err = z.Resources.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "tmpfs":
			var zcua uint32
			zcua, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				return
			}
			if z.Tmpfs == nil && zcua > 0 {
				z.Tmpfs = make(map[string]string, zcua)
			} else if len(z.Tmpfs) > 0 {
				for key, _ := range z.Tmpfs {
					delete(z.Tmpfs, key)
				}
			}
			for zcua > 0 {
				var zxvk string
				var zbzg string
				zcua--
				zxvk, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				zbzg, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				z.Tmpfs[zxvk] = zbzg
			}
		case "binds":
			var zxhx uint32
			zxhx, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Binds) >= int(zxhx) {
				z.Binds = (z.Binds)[:zxhx]
			} else {
				z.Binds = make([]string, zxhx)
			}
			for zbai := range z.Binds {
				z.Binds[zbai], bts, err = msgp.ReadStringBytes(bts)
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
	s = 1 + 9 + msgp.StringPrefixSize + len(z.Registry) + 11 + msgp.StringPrefixSize + len(z.Repository) + 9 + msgp.StringPrefixSize + len(z.Endpoint) + 13 + msgp.StringPrefixSize + len(z.NetworkMode) + 13 + msgp.StringPrefixSize + len(z.RuntimePath) + 4 + msgp.StringPrefixSize + len(z.Cwd) + 10 + z.Resources.Msgsize() + 6 + msgp.MapHeaderSize
	if z.Tmpfs != nil {
		for zxvk, zbzg := range z.Tmpfs {
			_ = zbzg
			s += msgp.StringPrefixSize + len(zxvk) + msgp.StringPrefixSize + len(zbzg)
		}
	}
	s += 6 + msgp.ArrayHeaderSize
	for zbai := range z.Binds {
		s += msgp.StringPrefixSize + len(z.Binds[zbai])
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Resources) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zlqf uint32
	zlqf, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zlqf > 0 {
		zlqf--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "memory":
			err = z.Memory.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "CpuShares":
			err = z.CPUShares.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "CpuPeriod":
			err = z.CPUPeriod.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "CpuQuota":
			err = z.CPUQuota.DecodeMsg(dc)
			if err != nil {
				return
			}
		case "CpusetCpus":
			z.CpusetCpus, err = dc.ReadString()
			if err != nil {
				return
			}
		case "CpusetMems":
			z.CpusetMems, err = dc.ReadString()
			if err != nil {
				return
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
func (z *Resources) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 6
	// write "memory"
	err = en.Append(0x86, 0xa6, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79)
	if err != nil {
		return err
	}
	err = z.Memory.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "CpuShares"
	err = en.Append(0xa9, 0x43, 0x70, 0x75, 0x53, 0x68, 0x61, 0x72, 0x65, 0x73)
	if err != nil {
		return err
	}
	err = z.CPUShares.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "CpuPeriod"
	err = en.Append(0xa9, 0x43, 0x70, 0x75, 0x50, 0x65, 0x72, 0x69, 0x6f, 0x64)
	if err != nil {
		return err
	}
	err = z.CPUPeriod.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "CpuQuota"
	err = en.Append(0xa8, 0x43, 0x70, 0x75, 0x51, 0x75, 0x6f, 0x74, 0x61)
	if err != nil {
		return err
	}
	err = z.CPUQuota.EncodeMsg(en)
	if err != nil {
		return
	}
	// write "CpusetCpus"
	err = en.Append(0xaa, 0x43, 0x70, 0x75, 0x73, 0x65, 0x74, 0x43, 0x70, 0x75, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteString(z.CpusetCpus)
	if err != nil {
		return
	}
	// write "CpusetMems"
	err = en.Append(0xaa, 0x43, 0x70, 0x75, 0x73, 0x65, 0x74, 0x4d, 0x65, 0x6d, 0x73)
	if err != nil {
		return err
	}
	err = en.WriteString(z.CpusetMems)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Resources) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 6
	// string "memory"
	o = append(o, 0x86, 0xa6, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79)
	o, err = z.Memory.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "CpuShares"
	o = append(o, 0xa9, 0x43, 0x70, 0x75, 0x53, 0x68, 0x61, 0x72, 0x65, 0x73)
	o, err = z.CPUShares.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "CpuPeriod"
	o = append(o, 0xa9, 0x43, 0x70, 0x75, 0x50, 0x65, 0x72, 0x69, 0x6f, 0x64)
	o, err = z.CPUPeriod.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "CpuQuota"
	o = append(o, 0xa8, 0x43, 0x70, 0x75, 0x51, 0x75, 0x6f, 0x74, 0x61)
	o, err = z.CPUQuota.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "CpusetCpus"
	o = append(o, 0xaa, 0x43, 0x70, 0x75, 0x73, 0x65, 0x74, 0x43, 0x70, 0x75, 0x73)
	o = msgp.AppendString(o, z.CpusetCpus)
	// string "CpusetMems"
	o = append(o, 0xaa, 0x43, 0x70, 0x75, 0x73, 0x65, 0x74, 0x4d, 0x65, 0x6d, 0x73)
	o = msgp.AppendString(o, z.CpusetMems)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Resources) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zdaf uint32
	zdaf, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zdaf > 0 {
		zdaf--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "memory":
			bts, err = z.Memory.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "CpuShares":
			bts, err = z.CPUShares.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "CpuPeriod":
			bts, err = z.CPUPeriod.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "CpuQuota":
			bts, err = z.CPUQuota.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "CpusetCpus":
			z.CpusetCpus, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "CpusetMems":
			z.CpusetMems, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
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
func (z *Resources) Msgsize() (s int) {
	s = 1 + 7 + z.Memory.Msgsize() + 10 + z.CPUShares.Msgsize() + 10 + z.CPUPeriod.Msgsize() + 9 + z.CPUQuota.Msgsize() + 11 + msgp.StringPrefixSize + len(z.CpusetCpus) + 11 + msgp.StringPrefixSize + len(z.CpusetMems)
	return
}
