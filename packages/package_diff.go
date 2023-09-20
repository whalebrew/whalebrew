package packages

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/google/go-cmp/cmp"
)

var ErrNotExhaustiveChangeTypeSwitch = errors.New("change type switch is not exhaustive this is likely a programming error")

type fieldChangeReporter func(change StructChange) string

type StructChange interface {
	FieldName() string
}

type Addition struct {
	field      string
	AddedValue reflect.Value
}

func (a Addition) FieldName() string {
	return a.field
}

type Removal struct {
	field        string
	RemovedValue reflect.Value
}

func (r Removal) FieldName() string {
	return r.field
}

type Modification struct {
	fieldName string
	FieldType reflect.Type
	PrevValue reflect.Value
	CurrValue reflect.Value
}

func (m Modification) FieldName() string {
	return m.fieldName
}

func genericFieldReporter(change StructChange) string {
	var val reflect.Value
	var changePrefix rune
	switch v := change.(type) {
	case Modification:
		return fmt.Sprintf("> %v to %v", v.PrevValue, v.CurrValue)
	case Addition:
		val = v.AddedValue
		changePrefix = '+'
	case Removal:
		val = v.RemovedValue
		changePrefix = '-'
	default:
		panic(ErrNotExhaustiveChangeTypeSwitch)
	}

	return writeValue(val, func(value reflect.Value, out *strings.Builder) {
		fmt.Fprintf(out, "%c %v", changePrefix, value)
	})
}

var permissionChangeIterationOrder = []string{"Environment", "Ports", "Volumes"}
var permissionChangeReporters = map[string]fieldChangeReporter{
	"Ports": func(change StructChange) string {
		var port reflect.Value

		var msg strings.Builder

		switch v := change.(type) {
		case Modification:
			prevPort := v.PrevValue.String()
			currPort := v.CurrValue.String()

			isPrevUdp := strings.HasSuffix(prevPort, "udp")
			isCurrUdp := strings.HasSuffix(currPort, "udp")

			msg.WriteString("* Listen on")
			if isPrevUdp && !isCurrUdp {
				msg.WriteString(" changed from UDP to TCP")
			} else if !isPrevUdp && isCurrUdp {
				msg.WriteString(" changed from TCP to UDP")
			} else if isCurrUdp {
				msg.WriteString(" UDP")
			} else {
				msg.WriteString(" TCP")
			}

			prevPort = strings.Split(prevPort, ":")[0]
			currPort = strings.Split(currPort, ":")[0]

			if prevPort != currPort {
				fmt.Fprintf(&msg, " port changed from %s to %s", prevPort, currPort)
			} else {
				fmt.Fprintf(&msg, " port %s", currPort)
			}

			return msg.String()

		case Addition:
			port = v.AddedValue
		case Removal:
			port = v.RemovedValue
		default:
			panic(ErrNotExhaustiveChangeTypeSwitch)
		}

		return writeValue(port, func(value reflect.Value, out *strings.Builder) {
			portS := value.String()

			portNumber := strings.Split(portS, ":")[0]
			proto := "TCP"
			if strings.HasSuffix(portS, "udp") {
				proto = "UDP"
			}

			fmt.Fprintf(out, "* Listen on %s port %s", proto, portNumber)
		})
	},
	"Environment": func(change StructChange) string {
		var envVar reflect.Value

		switch v := change.(type) {
		case Modification:
			return fmt.Sprintf("* Read the environment variable %s instead of %s", v.CurrValue.String(), v.PrevValue.String())
		case Addition:
			envVar = v.AddedValue
		case Removal:
			envVar = v.RemovedValue
		default:
			panic(ErrNotExhaustiveChangeTypeSwitch)
		}

		return writeValue(envVar, func(value reflect.Value, out *strings.Builder) {
			fmt.Fprintf(out, "* Read the environment variable %s", value.String())
		})
	},
	"Volumes": func(change StructChange) string {
		var msg strings.Builder

		var volume reflect.Value

		switch v := change.(type) {
		case Modification:
			prevVolumeS := v.PrevValue.String()
			currVolumeS := v.CurrValue.String()

			prevParts := strings.Split(prevVolumeS, ":")
			currParts := strings.Split(currVolumeS, ":")

			if len(prevParts) > 1 && len(currParts) > 1 {
				prevRo := strings.HasSuffix(prevVolumeS, "ro")
				currRo := strings.HasSuffix(currVolumeS, "ro")

				toRw := prevRo && !currRo
				toRo := currRo && !prevRo

				if toRw {
					fmt.Fprint(&msg, "* Read changed to read and write of the file or directory")
				} else if toRo {
					fmt.Fprint(&msg, "* Read and write changed to read of the file or directory")
				} else if currRo {
					fmt.Fprint(&msg, "* Read the file or directory")
				} else {
					fmt.Fprint(&msg, "* Read and write to the file or directory")
				}

				if prevParts[0] != currParts[0] {
					fmt.Fprintf(&msg, " changed from %q to %q", prevParts[0], currParts[0])
				} else {
					fmt.Fprintf(&msg, " %q", currParts[0])
				}
			}

			return msg.String()

		case Addition:
			volume = v.AddedValue
		case Removal:
			volume = v.RemovedValue
		default:
			panic(ErrNotExhaustiveChangeTypeSwitch)
		}

		return writeValue(volume, func(value reflect.Value, out *strings.Builder) {
			volumeS := value.String()
			parts := strings.Split(volumeS, ":")

			if len(parts) > 1 {
				if strings.HasSuffix(volumeS, "ro") {
					fmt.Fprintf(out, "* Read the file or directory %q", parts[0])
				} else {
					fmt.Fprintf(out, "* Read and write to the file or directory %q", parts[0])
				}
			}
		})
	},
}

func writeValue(val reflect.Value, printer func(reflect.Value, *strings.Builder)) string {
	var msg strings.Builder
	if isIndexableType(val.Type()) {
		totalElements := val.Len()
		for i := 0; i < totalElements; i++ {
			element := val.Index(i)

			printer(element, &msg)

			if i < totalElements-1 {
				msg.WriteRune('\n')
			}
		}
	} else {
		printer(val, &msg)
	}

	return msg.String()
}

func isIndexableType(t reflect.Type) bool {
	return t.Kind() == reflect.Slice || t.Kind() == reflect.Array
}

type DiffReporter struct {
	path  cmp.Path
	diffs map[string][]StructChange
}

func NewDiffReporter() *DiffReporter {
	return &DiffReporter{diffs: make(map[string][]StructChange)}
}

func (r *DiffReporter) PushStep(ps cmp.PathStep) {
	r.path = append(r.path, ps)
}

func reportStructChange(path cmp.Path) StructChange {
	leaf := path.Last()
	vx, vy := leaf.Values()

	var change StructChange

	switch v := leaf.(type) {
	case cmp.SliceIndex:
		parent := path.Index(-2)

		if _, ok := parent.(cmp.StructField); !ok {
			panic(errors.New("can not compare with non struct type root value"))
		}

		fieldName := strings.TrimLeft(parent.String(), ".")

		ix, iy := v.SplitKeys()

		switch {
		case ix == iy:
			change = Modification{
				fieldName: fieldName,
				FieldType: parent.Type(),
				PrevValue: vx,
				CurrValue: vy,
			}
		case iy == -1:
			// [5->?] means "I don't know where X[5] went"
			change = Removal{
				field:        fieldName,
				RemovedValue: vx,
			}
		case ix == -1:
			// [?->3] means "I don't know where Y[3] came from"
			change = Addition{
				field:      fieldName,
				AddedValue: vy,
			}
		}

	case cmp.StructField:
		fieldName := strings.TrimLeft(leaf.String(), ".")

		if !vx.IsValid() || (isIndexableType(vx.Type()) && isIndexableType(vy.Type()) && vx.Len() == 0 && vy.Len() > 0) {
			change = Addition{
				field:      fieldName,
				AddedValue: vy,
			}
		} else if !vy.IsValid() || (isIndexableType(vy.Type()) && isIndexableType(vx.Type()) && vy.Len() == 0 && vx.Len() > 0) {
			change = Removal{
				field:        fieldName,
				RemovedValue: vx,
			}
		} else {
			change = Modification{
				fieldName: fieldName,
				FieldType: leaf.Type(),
				PrevValue: vx,
				CurrValue: vy,
			}
		}
	}

	return change
}

func (r *DiffReporter) Report(rs cmp.Result) {
	if !rs.Equal() {
		change := reportStructChange(r.path)

		if change != nil {
			r.diffs[change.FieldName()] = append(r.diffs[change.FieldName()], change)
		}
	}
}

func (r *DiffReporter) PopStep() {
	r.path = r.path[:len(r.path)-1]
}

func (r *DiffReporter) String() string {
	var additions, removals, modifications strings.Builder

	for field, changes := range r.diffs {
		var wroteHeader [3]bool

		for _, change := range changes {
			var changeReporter fieldChangeReporter = genericFieldReporter

			switch c := change.(type) {
			case Addition:
				if !wroteHeader[0] {
					fmt.Fprintf(&additions, "Added these values to %s\n", field)
					wroteHeader[0] = true
				}
				fmt.Fprintf(&additions, "%s\n", changeReporter(change))
			case Removal:
				if !wroteHeader[1] {
					fmt.Fprintf(&removals, "Removed these values from %s\n", field)
					wroteHeader[1] = true
				}
				fmt.Fprintf(&removals, "%s\n", changeReporter(change))
			case Modification:
				if isIndexableType(c.FieldType) {
					if !wroteHeader[2] {
						fmt.Fprintf(&modifications, "Modified these items of %s\n", field)
						wroteHeader[2] = true
					}
					fmt.Fprintf(&modifications, "%s\n", changeReporter(change))
				} else {
					fmt.Fprintf(&modifications, "Modified %s\n%s", field, changeReporter(change))
				}
			}
		}
	}

	var result []string
	if additions.Len() > 0 {
		result = append(result, additions.String())
	}

	if removals.Len() > 0 {
		result = append(result, removals.String())
	}

	if modifications.Len() > 0 {
		result = append(result, modifications.String())
	}

	return strings.Join(result, "\n")
}

type PermissionChangeReporter struct {
	path         cmp.Path
	diffs        map[string][]StructChange
	freshInstall bool
}

func NewPermissionChangeReporter(freshInstall bool) *PermissionChangeReporter {
	return &PermissionChangeReporter{diffs: make(map[string][]StructChange), freshInstall: freshInstall}
}

func (r *PermissionChangeReporter) Report(rs cmp.Result) {
	if !rs.Equal() {
		change := reportStructChange(r.path)

		if change != nil {
			r.diffs[change.FieldName()] = append(r.diffs[change.FieldName()], change)
		}
	}
}

func (r *PermissionChangeReporter) PushStep(ps cmp.PathStep) {
	r.path = append(r.path, ps)
}

func (r *PermissionChangeReporter) PopStep() {
	r.path = r.path[:len(r.path)-1]
}

func (r *PermissionChangeReporter) String() string {
	var additions, removals, modifications strings.Builder

	var wroteHeader [3]bool
	for _, field := range permissionChangeIterationOrder {
		changes, ok := r.diffs[field]
		if !ok {
			continue
		}

		for _, change := range changes {
			if changeReporter, ok := permissionChangeReporters[field]; ok {
				switch change.(type) {
				case Addition:
					if !wroteHeader[0] {
						if r.freshInstall {
							fmt.Fprint(&additions, "This package needs additional access to your system. It wants to:\n\n")
						} else {
							fmt.Fprint(&additions, "This package update requests additional permissions to the ones it currently has:\n\n")
						}
						wroteHeader[0] = true
					}
					fmt.Fprintf(&additions, "%s\n", changeReporter(change))
				case Removal:
					if !wroteHeader[1] {
						fmt.Fprint(&removals, "Updating this package will remove some of its current permissions:\n\n")
						wroteHeader[1] = true
					}
					fmt.Fprintf(&removals, "%s\n", changeReporter(change))
				case Modification:
					if !wroteHeader[2] {
						fmt.Fprint(&modifications, "Updating this package will modify some of its current permissions\n\n")
						wroteHeader[2] = true
					}
					fmt.Fprintf(&modifications, "%s\n", changeReporter(change))
				}
			}
		}
	}

	var result []string
	if additions.Len() > 0 {
		result = append(result, additions.String())
	}

	if removals.Len() > 0 {
		result = append(result, removals.String())
	}

	if modifications.Len() > 0 {
		result = append(result, modifications.String())
	}

	return strings.Join(result, "\n")
}
