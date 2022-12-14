package archer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Faire/archer/lib/archer/utils"
)

type Project struct {
	Root      string
	Name      string
	NameParts []string
	Type      ProjectType

	RootDir     string
	Dir         string
	ProjectFile string

	dependencies map[string]*Dependency
	size         map[string]Size
	config       map[string]string

	dataDir string
}

func NewProject(root, name string) *Project {
	return &Project{
		Root:         root,
		Name:         name,
		dependencies: map[string]*Dependency{},
		size:         map[string]Size{},
		config:       map[string]string{},
	}
}

func (p *Project) String() string {
	return fmt.Sprintf("%v[%v]", p.Name, p.Type)
}

func (p *Project) AddDependency(d *Project) *Dependency {
	result := &Dependency{
		Source: p,
		Target: d,
		config: map[string]string{},
	}

	p.dependencies[d.Name] = result

	return result
}

func (p *Project) AddSize(name string, size Size) {
	p.size[name] = size
}

func (p *Project) GetSize() Size {
	result := Size{
		Other: map[string]int{},
	}

	for _, v := range p.size {
		result.Add(v)
	}

	return result
}

func (p *Project) GetSizeOf(name string) Size {
	result, ok := p.size[name]

	if !ok {
		result = Size{
			Other: map[string]int{},
		}
	}

	return result
}

func (p *Project) FullName() string {
	return p.Root + ":" + p.Name
}

func (p *Project) SimpleName() string {
	return p.LevelSimpleName(0)
}

func (p *Project) LevelSimpleName(level int) string {
	if len(p.NameParts) == 0 {
		return p.Name
	}

	parts := p.NameParts

	if level > 0 {
		parts = utils.Take(parts, level)
	}

	parts = simplifyPrefixes(parts)

	result := strings.Join(parts, ":")

	if len(p.Name) <= len(result) {
		result = p.Name
	}

	return result
}

func simplifyPrefixes(parts []string) []string {
	for len(parts) > 1 && strings.HasPrefix(parts[1], parts[0]) {
		parts = parts[1:]
	}
	return parts
}

func (p *Project) IsIgnored() bool {
	return utils.IsTrue(p.GetConfig("ignore"))
}

func (p *Project) IsCode() bool {
	return p.Type == CodeType
}

func (p *Project) IsExternalDependency() bool {
	return p.Type == ExternalDependencyType
}

func (p *Project) ListDependencies(filter FilterType) []*Dependency {
	result := make([]*Dependency, 0, len(p.dependencies))

	for _, v := range p.dependencies {
		if filter == FilterExcludeExternal && v.Target.IsExternalDependency() {
			continue
		}

		result = append(result, v)
	}

	sortDependencies(result)

	return result
}

func sortDependencies(result []*Dependency) {
	sort.Slice(result, func(i, j int) bool {
		pi := result[i].Source
		pj := result[j].Source

		if pi.Name == pj.Name {
			pi = result[i].Target
			pj = result[j].Target
		}

		if pi.IsCode() && pj.IsExternalDependency() {
			return true
		}

		if pi.IsExternalDependency() && pj.IsCode() {
			return false
		}

		return strings.TrimLeft(pi.Name, ":") < strings.TrimLeft(pj.Name, ":")
	})
}

func (p *Project) SetConfig(config string, value string) bool {
	if p.GetConfig(config) == value {
		return false
	}

	if value == "" {
		delete(p.config, config)
	} else {
		p.config[config] = value
	}

	return true
}

func (p *Project) GetConfig(config string) string {
	v, _ := p.config[config]
	return v
}

type Dependency struct {
	Source *Project
	Target *Project
	config map[string]string
}

func (d *Dependency) String() string {
	return fmt.Sprintf("%v -> %v", d.Source, d.Target)
}

func (d *Dependency) SetConfig(config string, value string) bool {
	if d.GetConfig(config) == value {
		return false
	}

	if value == "" {
		delete(d.config, config)
	} else {
		d.config[config] = value
	}

	return true
}

func (d *Dependency) GetConfig(config string) string {
	v, _ := d.config[config]
	return v
}

type Projects struct {
	all map[string]*Project
}

func NewProjects() *Projects {
	return &Projects{
		all: map[string]*Project{},
	}
}

func (ps *Projects) GetOrNil(name string) *Project {
	if len(name) == 0 {
		panic("empty name not supported")
	}

	result, ok := ps.all[name]
	if !ok {
		return nil
	}

	return result
}

func (ps *Projects) Get(root, name string) *Project {
	if len(root) == 0 {
		panic("empty root not supported")
	}
	if len(name) == 0 {
		panic("empty name not supported")
	}

	key := root + "\n" + name
	result, ok := ps.all[key]

	if !ok {
		result = NewProject(root, name)
		ps.all[key] = result
	}

	return result
}

func (ps *Projects) ListProjects(filter FilterType) []*Project {
	result := make([]*Project, 0, len(ps.all))

	for _, v := range ps.all {
		if filter == FilterExcludeExternal && v.IsExternalDependency() {
			continue
		}

		result = append(result, v)
	}

	sortProjects(result)

	return result
}

func sortProjects(result []*Project) {
	sort.Slice(result, func(i, j int) bool {
		pi := result[i]
		pj := result[j]

		if pi.IsCode() && pj.IsExternalDependency() {
			return true
		}

		if pi.IsExternalDependency() && pj.IsCode() {
			return false
		}

		return strings.TrimLeft(pi.Name, ":") < strings.TrimLeft(pj.Name, ":")
	})
}

type Size struct {
	Lines int
	Files int
	Bytes int
	Other map[string]int
}

func (l *Size) Add(other Size) {
	l.Lines += other.Lines
	l.Files += other.Files
	l.Bytes += other.Bytes

	for k, v := range other.Other {
		o, _ := l.Other[k]
		l.Other[k] = o + v
	}
}

type FilterType int

const (
	FilterAll FilterType = iota
	FilterExcludeExternal
)

type ProjectType int

const (
	ExternalDependencyType ProjectType = iota
	CodeType
	DatabaseType
)
