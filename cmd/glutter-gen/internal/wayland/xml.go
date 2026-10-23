package wayland

import (
	"encoding/xml"

	"github.com/rm4n0s/errors"
)

// The types below are the XML of a Wayland protocol description, as
// wayland.dtd defines it.

type protocol struct {
	XMLName    xml.Name `xml:"protocol"`
	Name       string   `xml:"name,attr"`
	Copyright  string   `xml:"copyright"`
	Interfaces []*iface `xml:"interface"`
	file       string   // base name of the XML file
}

type description struct {
	Summary string `xml:"summary,attr"`
	Text    string `xml:",chardata"`
}

type iface struct {
	Name        string      `xml:"name,attr"`
	Version     int         `xml:"version,attr"`
	Description description `xml:"description"`
	Requests    []*message  `xml:"request"`
	Events      []*message  `xml:"event"`
	Enums       []*enum     `xml:"enum"`
	proto       *protocol
}

type message struct {
	Name        string      `xml:"name,attr"`
	Type        string      `xml:"type,attr"`
	Since       int         `xml:"since,attr"`
	Description description `xml:"description"`
	Args        []*arg      `xml:"arg"`
}

type arg struct {
	Name      string `xml:"name,attr"`
	Type      string `xml:"type,attr"`
	Interface string `xml:"interface,attr"`
	AllowNull bool   `xml:"allow-null,attr"`
	Enum      string `xml:"enum,attr"`
	Summary   string `xml:"summary,attr"`
}

type enum struct {
	Name        string      `xml:"name,attr"`
	Bitfield    bool        `xml:"bitfield,attr"`
	Description description `xml:"description"`
	Entries     []*entry    `xml:"entry"`
}

type entry struct {
	Name    string `xml:"name,attr"`
	Value   string `xml:"value,attr"`
	Summary string `xml:"summary,attr"`
}

// argument types of wayland.dtd
const (
	typeInt    = "int"
	typeUint   = "uint"
	typeFixed  = "fixed"
	typeString = "string"
	typeObject = "object"
	typeNewID  = "new_id"
	typeArray  = "array"
	typeFD     = "fd"
)

// parseProtocol reads one protocol description. file is its base name.
func parseProtocol(data []byte, file string) (*protocol, error) {
	p := new(protocol)
	if err := xml.Unmarshal(data, p); err != nil {
		return nil, errors.NewErr(tagWaylandXMLParseFailed, err).SetMetadata("file", file)
	}
	p.file = file
	for _, i := range p.Interfaces {
		i.proto = p
		for _, m := range append(append([]*message(nil), i.Requests...), i.Events...) {
			if m.Since == 0 {
				m.Since = 1
			}
		}
	}
	return p, nil
}

// registry finds interfaces and enums across all the protocols, because the
// protocols refer to each other.
type registry struct {
	protocols []*protocol
	ifaces    map[string]*iface
}

// newRegistry indexes the protocols and checks everything the generator relies
// on: that interface names are unique, that every referenced interface and enum
// exists, and that argument types are known. It fails with WaylandXMLInvalid.
func newRegistry(protocols []*protocol) (*registry, error) {
	r := &registry{protocols: protocols, ifaces: map[string]*iface{}}
	for _, p := range protocols {
		for _, i := range p.Interfaces {
			if other, dup := r.ifaces[i.Name]; dup {
				return nil, errors.New(tagWaylandXMLInvalid, "interface defined twice",
					"interface", i.Name, "file", p.file, "otherFile", other.proto.file)
			}
			r.ifaces[i.Name] = i
		}
	}
	for _, p := range protocols {
		for _, i := range p.Interfaces {
			if err := r.check(i); err != nil {
				return nil, err
			}
		}
	}
	return r, nil
}

func (r *registry) check(i *iface) error {
	for _, kind := range []struct {
		name string
		msgs []*message
	}{{"request", i.Requests}, {"event", i.Events}} {
		for _, m := range kind.msgs {
			newIDs := 0
			for _, a := range m.Args {
				where := []any{"file", i.proto.file, "interface", i.Name, kind.name, m.Name, "arg", a.Name}
				switch a.Type {
				case typeInt, typeUint, typeFixed, typeString, typeArray, typeFD:
				case typeObject, typeNewID:
					if a.Interface != "" {
						if _, ok := r.ifaces[a.Interface]; !ok {
							return errors.New(tagWaylandXMLInvalid, "argument refers to an interface that no protocol defines",
								append(where, "refers", a.Interface)...)
						}
					}
					if a.Type == typeNewID {
						newIDs++
					}
				default:
					return errors.New(tagWaylandXMLInvalid, "unknown argument type", append(where, "type", a.Type)...)
				}
				if a.Enum != "" {
					if a.Type != typeInt && a.Type != typeUint {
						return errors.New(tagWaylandXMLInvalid, "only int and uint arguments can have an enum", where...)
					}
					if _, _, ok := r.findEnum(i, a.Enum); !ok {
						return errors.New(tagWaylandXMLInvalid, "argument refers to an enum that no protocol defines",
							append(where, "enum", a.Enum)...)
					}
				}
			}
			if newIDs > 1 {
				return errors.New(tagWaylandXMLInvalid, "a message with more than one new_id argument is not supported",
					"file", i.proto.file, "interface", i.Name, kind.name, m.Name)
			}
			if kind.name == "request" && m.Type != "" && m.Type != "destructor" {
				return errors.New(tagWaylandXMLInvalid, "unknown request type",
					"file", i.proto.file, "interface", i.Name, "request", m.Name, "type", m.Type)
			}
		}
	}
	return nil
}

// findEnum resolves the enum attribute of an argument of interface from: either
// "name" in the same interface or "interface.name" in another.
func (r *registry) findEnum(from *iface, ref string) (owner *iface, e *enum, ok bool) {
	ifaceName, enumName := from.Name, ref
	for k := 0; k < len(ref); k++ {
		if ref[k] == '.' {
			ifaceName, enumName = ref[:k], ref[k+1:]
			break
		}
	}
	owner, found := r.ifaces[ifaceName]
	if !found {
		return nil, nil, false
	}
	for _, e := range owner.Enums {
		if e.Name == enumName {
			return owner, e, true
		}
	}
	return nil, nil, false
}

// The tags of the errors that this file makes (rm4n0s/errors, R5).
const (
	tagWaylandXMLInvalid     = "WaylandXMLInvalid"
	tagWaylandXMLParseFailed = "WaylandXMLParseFailed"
)
