// Code generated by "libovsdb.modelgen"
// DO NOT EDIT.

package vswitchd

import "github.com/ovn-org/libovsdb/model"

const DatapathTable = "Datapath"

// Datapath defines an object in Datapath table
type Datapath struct {
	UUID            string            `ovsdb:"_uuid"`
	Capabilities    map[string]string `ovsdb:"capabilities"`
	CTZones         map[int]string    `ovsdb:"ct_zones"`
	DatapathVersion string            `ovsdb:"datapath_version"`
	ExternalIDs     map[string]string `ovsdb:"external_ids"`
}

func (a *Datapath) GetUUID() string {
	return a.UUID
}

func (a *Datapath) GetCapabilities() map[string]string {
	return a.Capabilities
}

func copyDatapathCapabilities(a map[string]string) map[string]string {
	if a == nil {
		return nil
	}
	b := make(map[string]string, len(a))
	for k, v := range a {
		b[k] = v
	}
	return b
}

func equalDatapathCapabilities(a, b map[string]string) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || v != w {
			return false
		}
	}
	return true
}

func (a *Datapath) GetCTZones() map[int]string {
	return a.CTZones
}

func copyDatapathCTZones(a map[int]string) map[int]string {
	if a == nil {
		return nil
	}
	b := make(map[int]string, len(a))
	for k, v := range a {
		b[k] = v
	}
	return b
}

func equalDatapathCTZones(a, b map[int]string) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || v != w {
			return false
		}
	}
	return true
}

func (a *Datapath) GetDatapathVersion() string {
	return a.DatapathVersion
}

func (a *Datapath) GetExternalIDs() map[string]string {
	return a.ExternalIDs
}

func copyDatapathExternalIDs(a map[string]string) map[string]string {
	if a == nil {
		return nil
	}
	b := make(map[string]string, len(a))
	for k, v := range a {
		b[k] = v
	}
	return b
}

func equalDatapathExternalIDs(a, b map[string]string) bool {
	if (a == nil) != (b == nil) {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || v != w {
			return false
		}
	}
	return true
}

func (a *Datapath) DeepCopyInto(b *Datapath) {
	*b = *a
	b.Capabilities = copyDatapathCapabilities(a.Capabilities)
	b.CTZones = copyDatapathCTZones(a.CTZones)
	b.ExternalIDs = copyDatapathExternalIDs(a.ExternalIDs)
}

func (a *Datapath) DeepCopy() *Datapath {
	b := new(Datapath)
	a.DeepCopyInto(b)
	return b
}

func (a *Datapath) CloneModelInto(b model.Model) {
	c := b.(*Datapath)
	a.DeepCopyInto(c)
}

func (a *Datapath) CloneModel() model.Model {
	return a.DeepCopy()
}

func (a *Datapath) Equals(b *Datapath) bool {
	return a.UUID == b.UUID &&
		equalDatapathCapabilities(a.Capabilities, b.Capabilities) &&
		equalDatapathCTZones(a.CTZones, b.CTZones) &&
		a.DatapathVersion == b.DatapathVersion &&
		equalDatapathExternalIDs(a.ExternalIDs, b.ExternalIDs)
}

func (a *Datapath) EqualsModel(b model.Model) bool {
	c := b.(*Datapath)
	return a.Equals(c)
}

var _ model.CloneableModel = &Datapath{}
var _ model.ComparableModel = &Datapath{}
