package isolate

// NOTE: THIS FILE WAS PRODUCED BY THE
// MSGP CODE GENERATION TOOL (github.com/tinylib/msgp)
// DO NOT EDIT

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *MarkedWorkerMetrics) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
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
func (z MarkedWorkerMetrics) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 0
	err = en.Append(0x80)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z MarkedWorkerMetrics) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 0
	o = append(o, 0x80)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MarkedWorkerMetrics) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
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
func (z MarkedWorkerMetrics) Msgsize() (s int) {
	s = 1
	return
}

// DecodeMsg implements msgp.Decodable
func (z *MetricsResponse) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0003 uint32
	zb0003, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	if (*z) == nil && zb0003 > 0 {
		(*z) = make(MetricsResponse, zb0003)
	} else if len((*z)) > 0 {
		for key, _ := range *z {
			delete((*z), key)
		}
	}
	for zb0003 > 0 {
		zb0003--
		var zb0001 string
		var zb0002 *WorkerMetrics
		zb0001, err = dc.ReadString()
		if err != nil {
			return
		}
		if dc.IsNil() {
			err = dc.ReadNil()
			if err != nil {
				return
			}
			zb0002 = nil
		} else {
			if zb0002 == nil {
				zb0002 = new(WorkerMetrics)
			}
			err = zb0002.DecodeMsg(dc)
			if err != nil {
				return
			}
		}
		(*z)[zb0001] = zb0002
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z MetricsResponse) EncodeMsg(en *msgp.Writer) (err error) {
	err = en.WriteMapHeader(uint32(len(z)))
	if err != nil {
		return
	}
	for zb0004, zb0005 := range z {
		err = en.WriteString(zb0004)
		if err != nil {
			return
		}
		if zb0005 == nil {
			err = en.WriteNil()
			if err != nil {
				return
			}
		} else {
			err = zb0005.EncodeMsg(en)
			if err != nil {
				return
			}
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z MetricsResponse) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendMapHeader(o, uint32(len(z)))
	for zb0004, zb0005 := range z {
		o = msgp.AppendString(o, zb0004)
		if zb0005 == nil {
			o = msgp.AppendNil(o)
		} else {
			o, err = zb0005.MarshalMsg(o)
			if err != nil {
				return
			}
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *MetricsResponse) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0003 uint32
	zb0003, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	if (*z) == nil && zb0003 > 0 {
		(*z) = make(MetricsResponse, zb0003)
	} else if len((*z)) > 0 {
		for key, _ := range *z {
			delete((*z), key)
		}
	}
	for zb0003 > 0 {
		var zb0001 string
		var zb0002 *WorkerMetrics
		zb0003--
		zb0001, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			return
		}
		if msgp.IsNil(bts) {
			bts, err = msgp.ReadNilBytes(bts)
			if err != nil {
				return
			}
			zb0002 = nil
		} else {
			if zb0002 == nil {
				zb0002 = new(WorkerMetrics)
			}
			bts, err = zb0002.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		}
		(*z)[zb0001] = zb0002
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z MetricsResponse) Msgsize() (s int) {
	s = msgp.MapHeaderSize
	if z != nil {
		for zb0004, zb0005 := range z {
			_ = zb0005
			s += msgp.StringPrefixSize + len(zb0004)
			if zb0005 == nil {
				s += msgp.NilSize
			} else {
				s += zb0005.Msgsize()
			}
		}
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *NetStat) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "rx_bytes":
			z.RxBytes, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "tx_bytes":
			z.TxBytes, err = dc.ReadUint64()
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
func (z NetStat) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "rx_bytes"
	err = en.Append(0x82, 0xa8, 0x72, 0x78, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.RxBytes)
	if err != nil {
		return
	}
	// write "tx_bytes"
	err = en.Append(0xa8, 0x74, 0x78, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.TxBytes)
	if err != nil {
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z NetStat) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "rx_bytes"
	o = append(o, 0x82, 0xa8, 0x72, 0x78, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	o = msgp.AppendUint64(o, z.RxBytes)
	// string "tx_bytes"
	o = append(o, 0xa8, 0x74, 0x78, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
	o = msgp.AppendUint64(o, z.TxBytes)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *NetStat) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "rx_bytes":
			z.RxBytes, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				return
			}
		case "tx_bytes":
			z.TxBytes, bts, err = msgp.ReadUint64Bytes(bts)
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
func (z NetStat) Msgsize() (s int) {
	s = 1 + 9 + msgp.Uint64Size + 9 + msgp.Uint64Size
	return
}

// DecodeMsg implements msgp.Decodable
func (z *WorkerMetrics) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "uptime":
			z.UptimeSec, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "cpu_usage":
			z.CpuUsageSec, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "cpu_load":
			z.CpuLoad, err = dc.ReadFloat32()
			if err != nil {
				return
			}
		case "mem":
			z.Mem, err = dc.ReadUint64()
			if err != nil {
				return
			}
		case "net":
			var zb0002 uint32
			zb0002, err = dc.ReadMapHeader()
			if err != nil {
				return
			}
			if z.Net == nil && zb0002 > 0 {
				z.Net = make(map[string]NetStat, zb0002)
			} else if len(z.Net) > 0 {
				for key, _ := range z.Net {
					delete(z.Net, key)
				}
			}
			for zb0002 > 0 {
				zb0002--
				var za0001 string
				var za0002 NetStat
				za0001, err = dc.ReadString()
				if err != nil {
					return
				}
				var zb0003 uint32
				zb0003, err = dc.ReadMapHeader()
				if err != nil {
					return
				}
				for zb0003 > 0 {
					zb0003--
					field, err = dc.ReadMapKeyPtr()
					if err != nil {
						return
					}
					switch msgp.UnsafeString(field) {
					case "rx_bytes":
						za0002.RxBytes, err = dc.ReadUint64()
						if err != nil {
							return
						}
					case "tx_bytes":
						za0002.TxBytes, err = dc.ReadUint64()
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
				z.Net[za0001] = za0002
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
func (z *WorkerMetrics) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 5
	// write "uptime"
	err = en.Append(0x85, 0xa6, 0x75, 0x70, 0x74, 0x69, 0x6d, 0x65)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.UptimeSec)
	if err != nil {
		return
	}
	// write "cpu_usage"
	err = en.Append(0xa9, 0x63, 0x70, 0x75, 0x5f, 0x75, 0x73, 0x61, 0x67, 0x65)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.CpuUsageSec)
	if err != nil {
		return
	}
	// write "cpu_load"
	err = en.Append(0xa8, 0x63, 0x70, 0x75, 0x5f, 0x6c, 0x6f, 0x61, 0x64)
	if err != nil {
		return
	}
	err = en.WriteFloat32(z.CpuLoad)
	if err != nil {
		return
	}
	// write "mem"
	err = en.Append(0xa3, 0x6d, 0x65, 0x6d)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.Mem)
	if err != nil {
		return
	}
	// write "net"
	err = en.Append(0xa3, 0x6e, 0x65, 0x74)
	if err != nil {
		return
	}
	err = en.WriteMapHeader(uint32(len(z.Net)))
	if err != nil {
		return
	}
	for za0001, za0002 := range z.Net {
		err = en.WriteString(za0001)
		if err != nil {
			return
		}
		// map header, size 2
		// write "rx_bytes"
		err = en.Append(0x82, 0xa8, 0x72, 0x78, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
		if err != nil {
			return
		}
		err = en.WriteUint64(za0002.RxBytes)
		if err != nil {
			return
		}
		// write "tx_bytes"
		err = en.Append(0xa8, 0x74, 0x78, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
		if err != nil {
			return
		}
		err = en.WriteUint64(za0002.TxBytes)
		if err != nil {
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *WorkerMetrics) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 5
	// string "uptime"
	o = append(o, 0x85, 0xa6, 0x75, 0x70, 0x74, 0x69, 0x6d, 0x65)
	o = msgp.AppendUint64(o, z.UptimeSec)
	// string "cpu_usage"
	o = append(o, 0xa9, 0x63, 0x70, 0x75, 0x5f, 0x75, 0x73, 0x61, 0x67, 0x65)
	o = msgp.AppendUint64(o, z.CpuUsageSec)
	// string "cpu_load"
	o = append(o, 0xa8, 0x63, 0x70, 0x75, 0x5f, 0x6c, 0x6f, 0x61, 0x64)
	o = msgp.AppendFloat32(o, z.CpuLoad)
	// string "mem"
	o = append(o, 0xa3, 0x6d, 0x65, 0x6d)
	o = msgp.AppendUint64(o, z.Mem)
	// string "net"
	o = append(o, 0xa3, 0x6e, 0x65, 0x74)
	o = msgp.AppendMapHeader(o, uint32(len(z.Net)))
	for za0001, za0002 := range z.Net {
		o = msgp.AppendString(o, za0001)
		// map header, size 2
		// string "rx_bytes"
		o = append(o, 0x82, 0xa8, 0x72, 0x78, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
		o = msgp.AppendUint64(o, za0002.RxBytes)
		// string "tx_bytes"
		o = append(o, 0xa8, 0x74, 0x78, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73)
		o = msgp.AppendUint64(o, za0002.TxBytes)
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *WorkerMetrics) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch msgp.UnsafeString(field) {
		case "uptime":
			z.UptimeSec, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				return
			}
		case "cpu_usage":
			z.CpuUsageSec, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				return
			}
		case "cpu_load":
			z.CpuLoad, bts, err = msgp.ReadFloat32Bytes(bts)
			if err != nil {
				return
			}
		case "mem":
			z.Mem, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				return
			}
		case "net":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				return
			}
			if z.Net == nil && zb0002 > 0 {
				z.Net = make(map[string]NetStat, zb0002)
			} else if len(z.Net) > 0 {
				for key, _ := range z.Net {
					delete(z.Net, key)
				}
			}
			for zb0002 > 0 {
				var za0001 string
				var za0002 NetStat
				zb0002--
				za0001, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
				var zb0003 uint32
				zb0003, bts, err = msgp.ReadMapHeaderBytes(bts)
				if err != nil {
					return
				}
				for zb0003 > 0 {
					zb0003--
					field, bts, err = msgp.ReadMapKeyZC(bts)
					if err != nil {
						return
					}
					switch msgp.UnsafeString(field) {
					case "rx_bytes":
						za0002.RxBytes, bts, err = msgp.ReadUint64Bytes(bts)
						if err != nil {
							return
						}
					case "tx_bytes":
						za0002.TxBytes, bts, err = msgp.ReadUint64Bytes(bts)
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
				z.Net[za0001] = za0002
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
func (z *WorkerMetrics) Msgsize() (s int) {
	s = 1 + 7 + msgp.Uint64Size + 10 + msgp.Uint64Size + 9 + msgp.Float32Size + 4 + msgp.Uint64Size + 4 + msgp.MapHeaderSize
	if z.Net != nil {
		for za0001, za0002 := range z.Net {
			_ = za0002
			s += msgp.StringPrefixSize + len(za0001) + 1 + 9 + msgp.Uint64Size + 9 + msgp.Uint64Size
		}
	}
	return
}
